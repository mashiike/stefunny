package stefunny

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/fatih/color"
	"github.com/mashiike/stefunny/asl"
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
		opts = append(opts, awsConfig.WithEndpointResolver(endpointsResolver))
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

func (app *App) Render(ctx context.Context, opt RenderOption) error {
	def, err := app.cfg.LoadDefinition()
	if err != nil {
		return err
	}
	stateMachine, err := asl.Parse(def)
	if err != nil {
		return err
	}
	bs, err := stateMachine.MarshalDOT(app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	_, err = opt.Writer.Write(bs)
	return err
}

func (app *App) LoadLoggingConfiguration(ctx context.Context) (*sfntypes.LoggingConfiguration, error) {
	ret := &sfntypes.LoggingConfiguration{
		Level: sfntypes.LogLevelOff,
	}
	cfg := app.cfg.StateMachine
	if cfg.Logging == nil {
		return ret, nil
	}
	if cfg.Logging.logLevel == sfntypes.LogLevelOff {
		return ret, nil
	}
	ret.IncludeExecutionData = *cfg.Logging.IncludeExecutionData
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

func (app *App) putSchedule(ctx context.Context, dryRun bool) error {
	if app.cfg.Schedule == nil {
		log.Println("[debug] schedule is not set")
		return nil
	}
	stateMachineArn, err := app.aws.GetStateMachineArn(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	ruleName := getScheduleRuleName(app.cfg.StateMachine.Name)
	putRuleInput := &eventbridge.PutRuleInput{
		Name:               &ruleName,
		Description:        aws.String(fmt.Sprintf("for state machine %s schedule", stateMachineArn)),
		ScheduleExpression: &app.cfg.Schedule.Expression,
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	}
	if output, err := app.aws.DescribeRule(ctx, &eventbridge.DescribeRuleInput{Name: &ruleName}); err == nil {
		if dryRun {
			var builder strings.Builder
			builder.WriteString(colorRestString(" {\n"))
			fmt.Fprintf(&builder, `   "Name":"%s",`+"\n", ruleName)
			if *putRuleInput.Description == *output.Description {
				fmt.Fprintf(&builder, `   "Description":"%s",`+"\n", *putRuleInput.Description)
			} else {
				fmt.Fprint(&builder, color.RedString(`-  "Description":"%s",`+"\n", *output.Description))
				fmt.Fprint(&builder, color.GreenString(`+  "Description":"%s,"`+"\n", *putRuleInput.Description))
			}
			if *putRuleInput.ScheduleExpression == *output.ScheduleExpression {
				fmt.Fprintf(&builder, `   "ScheduleExpression":"%s",`+"\n", *putRuleInput.ScheduleExpression)
			} else {
				fmt.Fprint(&builder, color.RedString(`-  "ScheduleExpression":"%s",`+"\n", *output.ScheduleExpression))
				fmt.Fprint(&builder, color.GreenString(`+  "ScheduleExpression":"%s",`+"\n", *putRuleInput.ScheduleExpression))
			}
			fmt.Fprintf(&builder, `   "State":"%s",`+"\n", output.State)
			fmt.Fprint(&builder, ` }`)

			log.Printf("[notice] update event bridge rule %s\n %s", dryRunStr, builder.String())
		} else {
			putRuleInput.State = output.State
		}
	} else {
		putRuleInput.State = eventbridgetypes.RuleStateEnabled
	}
	if dryRun {
		return nil
	}
	output, err := app.aws.PutRule(ctx, putRuleInput)
	if err != nil {
		return err
	}
	log.Printf("[notice] update event bridge rule arn `%s`", *output.RuleArn)
	return nil
}
