package stefunny

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny/internal/asl"
	"github.com/mashiike/stefunny/internal/jsonutil"
	"github.com/olekukonko/tablewriter"
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

func (app *App) Execute(ctx context.Context, opt ExecuteOption) error {
	dec := json.NewDecoder(opt.Stdin)
	var inputJSON interface{}
	if err := dec.Decode(&inputJSON); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(inputJSON, "", "  ")
	if err != nil {
		return err
	}
	input := string(bs)
	log.Printf("[info] input:\n%s\n", input)
	stateMachine, err := app.aws.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	if stateMachine.Type == sfntypes.StateMachineTypeExpress {
		return app.ExecuteForExpress(ctx, stateMachine, input, opt)
	}
	if stateMachine.Type == sfntypes.StateMachineTypeStandard {
		return app.ExecuteForStandard(ctx, stateMachine, input, opt)
	}
	return fmt.Errorf("unknown StateMachine Type:%s", stateMachine.Type)
}

func (app *App) ExecuteForExpress(ctx context.Context, stateMachine *StateMachine, input string, opt ExecuteOption) error {
	if opt.DumpHistory {
		log.Println("[warn] this state machine is EXPRESS type, history is not supported.")
	}
	if opt.Async {
		output, err := app.aws.StartExecution(ctx, stateMachine, opt.ExecutionName, input)
		if err != nil {
			return err
		}
		log.Printf("[notice] execution arn=%s", output.ExecutionArn)
		log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
		return nil
	}
	output, err := app.aws.StartSyncExecution(ctx, stateMachine, opt.ExecutionName, input)
	if err != nil {
		return err
	}
	log.Printf("[notice] execution arn=%s", *output.ExecutionArn)
	log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))

	if output.Status != sfntypes.SyncExecutionStatusSucceeded {
		if output.Error != nil {
			log.Println("[info] error: ", *output.Error)
		}
		if output.Cause != nil {
			log.Println("[info] cause: ", *output.Cause)
		}
		return errors.New("state machine execution failed")
	}
	log.Printf("[info] execution success")
	if opt.Stdout != nil && output.Output != nil {
		io.WriteString(opt.Stdout, *output.Output)
		io.WriteString(opt.Stdout, "\n")
	}
	return nil
}

func (app *App) ExecuteForStandard(ctx context.Context, stateMachine *StateMachine, input string, opt ExecuteOption) error {
	output, err := app.aws.StartExecution(ctx, stateMachine, opt.ExecutionName, input)
	if err != nil {
		return err
	}
	log.Printf("[notice] execution arn=%s", output.ExecutionArn)
	log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
	if opt.Async {
		return nil
	}
	waitOutput, err := app.aws.WaitExecution(ctx, output.ExecutionArn)
	if err != nil {
		return err
	}
	log.Printf("[info] execution time: %s", waitOutput.Elapsed())
	if opt.DumpHistory {
		events, err := app.aws.GetExecutionHistory(ctx, output.ExecutionArn)
		if err != nil {
			return err
		}
		table := tablewriter.NewWriter(opt.Stderr)
		table.SetHeader([]string{"ID", "Type", "Step", "Elapsed(ms)", "Timestamp"})
		for _, event := range events {
			table.Append([]string{
				fmt.Sprintf("%3d", event.Id),
				fmt.Sprintf("%v", event.HistoryEvent.Type),
				event.Step,
				fmt.Sprintf("%d", event.Elapsed().Milliseconds()),
				event.Timestamp.Format(time.RFC3339),
			})
		}
		table.Render()
	}
	if waitOutput.Datail != nil {
		log.Printf("[info] execution detail:\n%s", jsonutil.MarshalJSONString(waitOutput.Datail))
	}
	if waitOutput.Failed {
		return errors.New("state machine execution failed")
	}
	log.Printf("[info] execution success")
	if opt.Stdout != nil && len(waitOutput.Output) > 0 {
		io.WriteString(opt.Stdout, waitOutput.Output)
		io.WriteString(opt.Stdout, "\n")
	}
	return nil
}

func (app *App) Render(_ context.Context, opt RenderOption) error {
	def, err := app.cfg.LoadDefinition()
	if err != nil {
		return err
	}
	switch strings.ToLower(opt.Format) {
	case "", "dot":
		log.Println("[warn] dot format is deprecated (since v0.5.0)")
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
	case "json":
		_, err := io.WriteString(opt.Writer, def)
		return err
	case "yaml":
		log.Println("[warn] yaml format is deprecated (since v0.5.0)")
		bs, err := jsonutil.JSON2YAML([]byte(def))
		if err != nil {
			return err
		}
		_, err = opt.Writer.Write(bs)
		return err
	}
	return errors.New("unknown format")
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
