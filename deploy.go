package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/fatih/color"
)

func (app *App) Deploy(ctx context.Context, opt DeployOption) error {
	log.Println("[info] Starting deploy", opt.DryRunString())
	if err := app.deployStateMachine(ctx, opt); err != nil {
		return err
	}
	log.Println("[info] finish deploy", opt.DryRunString())
	return nil
}

func (app *App) deployStateMachine(ctx context.Context, opt DeployOption) error {
	newDefinition, err := app.cfg.LoadDefinition()
	if err != nil {
		return fmt.Errorf("can not load state machine definition: %w", err)
	}
	stateMachine, err := app.sfn.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}
	if stateMachine.Type != app.cfg.StateMachine.stateMachineType {
		return errors.New("state machine type is not match. replace state machine deploy not implemented")
	}

	if opt.DryRun {
		diffDef := jsonDiffString(*stateMachine.Definition, newDefinition)
		log.Printf(
			"[notice] change state machine definition %s\n\n%s\n",
			opt.DryRunString(),
			diffDef,
		)
	}
	input := &sfn.UpdateStateMachineInput{
		StateMachineArn: stateMachine.StateMachineArn,
		Definition:      &newDefinition,
	}
	if *stateMachine.RoleArn != app.cfg.StateMachine.RoleArn {
		if opt.DryRun {
			log.Printf(
				"[notice] change state machine role arn `%s`\n\n%s\n%s\n",
				opt.DryRunString(),
				color.RedString("-role_arn:%s", *stateMachine.RoleArn),
				color.GreenString("+role_arn:%s", app.cfg.StateMachine.RoleArn),
			)
		}
		input.RoleArn = &app.cfg.StateMachine.RoleArn
	}

	newLogging, err := app.LoadLoggingConfiguration(ctx)
	if err != nil {
		return err
	}
	logging := stateMachine.LoggingConfiguration
	if opt.DryRun {
		if logging.Level != newLogging.Level {
			log.Printf(
				"[notice] change state machine log level `%s`\n\n%s\n%s\n",
				opt.DryRunString(),
				color.RedString("-log_level:%s", logging.Level),
				color.GreenString("+log_level:%s", newLogging.Level),
			)
		}
		if logging.IncludeExecutionData != *app.cfg.StateMachine.Logging.IncludeExecutionData {
			log.Printf(
				"[notice] change state machine loogging.include_exection_data `%s`\n\n%s\n%s\n",
				opt.DryRunString(),
				color.RedString("-include_exection_data:%v", logging.IncludeExecutionData),
				color.GreenString("+include_exection_data:%v", *app.cfg.StateMachine.Logging.IncludeExecutionData),
			)
		}
		var changeDestinations bool
		if len(logging.Destinations) != len(newLogging.Destinations) {
			changeDestinations = true
		} else if len(logging.Destinations) != 0 {
			if *logging.Destinations[0].CloudWatchLogsLogGroup.LogGroupArn != *newLogging.Destinations[0].CloudWatchLogsLogGroup.LogGroupArn {
				changeDestinations = true
			}
		}
		if changeDestinations {
			log.Printf(
				"[notice] change state machine loogging.destinations `%s`\n\n%s\n%s\n",
				opt.DryRunString(),
				color.RedString("-destinations:%#v", marshalJSONString(logging.Destinations)),
				color.GreenString("+destinations:%#v", marshalJSONString(newLogging.Destinations)),
			)
		}
	}
	input.LoggingConfiguration = newLogging
	tracing := stateMachine.TracingConfiguration
	if tracing.Enabled != *app.cfg.StateMachine.Tracing.Enabled {
		if opt.DryRun {
			log.Printf(
				"[notice] change state machine tracing.enabled `%s`\n\n%s\n%s\n",
				opt.DryRunString(),
				color.RedString("-tracing.enabled:%v", tracing.Enabled),
				color.GreenString("+tracing.enabled:%v", *app.cfg.StateMachine.Tracing.Enabled),
			)
		}
		tracing.Enabled = *app.cfg.StateMachine.Tracing.Enabled
	}
	input.TracingConfiguration = tracing
	if opt.DryRun {
		return nil
	}
	output, err := app.sfn.UpdateStateMachine(ctx, input)
	if err != nil {
		return err
	}
	log.Printf("[info] updated state machine `%s`(at `%s`)\n", app.cfg.StateMachine.Name, *output.UpdateDate)
	return nil
}
