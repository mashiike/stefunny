package stefunny

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

const (
	tagManagedBy = "ManagedBy"
	appName      = "stefunny"
)

type App struct {
	cfg *Config
	aws *AWSService
}

func New(ctx context.Context, cfg *Config) (*App, error) {
	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(cfg.AWSRegion),
	}
	if endpointsResolver, ok := cfg.EndpointResolver(); ok {
		opts = append(opts, awsConfig.WithEndpointResolverWithOptions(endpointsResolver))
	}
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return NewWithClient(cfg, AWSClients{
		SFnClient:         sfn.NewFromConfig(awsCfg),
		CWLogsClient:      cloudwatchlogs.NewFromConfig(awsCfg),
		EventBridgeClient: eventbridge.NewFromConfig(awsCfg),
	})
}

func NewWithClient(cfg *Config, clients AWSClients) (*App, error) {
	return &App{
		cfg: cfg,
		aws: NewAWSService(clients),
	}, nil
}

func (app *App) LoadLoggingConfiguration(ctx context.Context) (*sfntypes.LoggingConfiguration, error) {
	ret := &sfntypes.LoggingConfiguration{
		Level: sfntypes.LogLevelOff,
	}
	cfg := app.cfg.StateMachine
	if cfg.Logging == nil {
		return ret, nil
	}
	ret.Level = cfg.Logging.logLevel
	ret.IncludeExecutionData = *cfg.Logging.IncludeExecutionData
	if cfg.Logging.Destination == nil {
		return ret, nil
	}
	arn, err := app.aws.GetLogGroupArn(ctx, cfg.Logging.Destination.LogGroup)
	if err != nil {
		return nil, fmt.Errorf("get log group arn: %w", err)
	}
	ret.Destinations = []sfntypes.LogDestination{
		{
			CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{
				LogGroupArn: &arn,
			},
		},
	}
	return ret, nil
}

func (app *App) LoadStateMachine(ctx context.Context) (*StateMachine, error) {
	definition, err := app.cfg.LoadDefinition()
	if err != nil {
		return nil, fmt.Errorf("load definition failed: %w", err)
	}
	logging, err := app.LoadLoggingConfiguration(ctx)
	if err != nil {
		return nil, fmt.Errorf("load logging config failed: %w", err)
	}
	stateMachine := &StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:                 &app.cfg.StateMachine.Name,
			Type:                 app.cfg.StateMachine.stateMachineType,
			RoleArn:              &app.cfg.StateMachine.RoleArn,
			Definition:           &definition,
			LoggingConfiguration: logging,
			TracingConfiguration: app.cfg.StateMachine.LoadTracingConfiguration(),
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String(tagManagedBy),
					Value: aws.String(appName),
				},
			},
		},
		Tags: app.cfg.Tags,
	}
	stateMachine.Tags[tagManagedBy] = appName
	return stateMachine, nil
}

func (app *App) LoadScheduleRules(_ context.Context, stateMachineArn string) (ScheduleRules, error) {
	rules := make([]*ScheduleRule, 0, len(app.cfg.Schedule))
	for _, cfg := range app.cfg.Schedule {
		rule := &ScheduleRule{
			PutRuleInput: eventbridge.PutRuleInput{
				Name:               aws.String(cfg.RuleName),
				ScheduleExpression: &cfg.Expression,
				State:              eventbridgetypes.RuleStateEnabled,
			},
			Targets: []eventbridgetypes.Target{{
				RoleArn: aws.String(cfg.RoleArn),
			}},
			TargetRoleArn: cfg.RoleArn,
			Tags:          app.cfg.Tags,
		}
		if cfg.Description != "" {
			rule.Description = aws.String(cfg.Description)
		}
		if cfg.ID != "" {
			rule.Targets[0].Id = aws.String(cfg.ID)
		}
		rule.Tags[tagManagedBy] = appName
		rule.SetStateMachineArn(stateMachineArn)
		rules = append(rules, rule)
	}
	return rules, nil
}
