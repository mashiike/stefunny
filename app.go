package stefunny

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
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

func (app *App) LoadStateMachine() (*StateMachine, error) {

	stateMachine := &StateMachine{
		CreateStateMachineInput: app.cfg.NewCreateStateMachineInput(),
		Tags:                    app.cfg.Tags,
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
