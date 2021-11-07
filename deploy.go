package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/fatih/color"
)

func (app *App) Deploy(ctx context.Context, opt DeployOption) error {
	log.Println("[info] Starting deploy", opt.DryRunString())
	if err := app.deployStateMachine(ctx, opt); err != nil {
		return err
	}
	if err := app.putSchedule(ctx, opt); err != nil {
		return err
	}
	log.Println("[info] finish deploy", opt.DryRunString())
	return nil
}

func (app *App) deployStateMachine(ctx context.Context, opt DeployOption) error {
	stateMachine, err := app.aws.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		log.Printf("[debug] %#v", err)
		if _, ok := err.(*sfntypes.StateMachineDoesNotExist); ok {
			return app.createStateMachine(ctx, opt)
		}
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}
	newDefinition, err := app.cfg.LoadDefinition()
	if err != nil {
		return fmt.Errorf("can not load state machine definition: %w", err)
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
	if tracing == nil {
		tracing = &sfntypes.TracingConfiguration{
			Enabled: false,
		}
	}
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
	output, err := app.aws.UpdateStateMachine(ctx, input)
	if err != nil {
		return err
	}
	log.Printf("[info] updated state machine `%s`(at `%s`)\n", app.cfg.StateMachine.Name, *output.UpdateDate)

	_, err = app.aws.TagResource(ctx, &sfn.TagResourceInput{
		ResourceArn: stateMachine.StateMachineArn,
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (app *App) putSchedule(ctx context.Context, opt DeployOption) error {
	if app.cfg.Schedule == nil {
		log.Println("[debug] schedule is not set")
		return nil
	}
	stateMachineArn, err := app.aws.GetStateMachineArn(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	ruleName := getScheduleRuleName(app.cfg.StateMachine.Name)
	if err := app.putEventBridgeRule(ctx, ruleName, stateMachineArn, opt); err != nil {
		return err
	}
	if err := app.putEventBridgeRuleTargets(ctx, ruleName, stateMachineArn, opt); err != nil {
		return err
	}

	return nil
}

func (app *App) putEventBridgeRule(ctx context.Context, ruleName, stateMachineArn string, opt DeployOption) error {
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
		if opt.DryRun {
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

			log.Printf("[notice] update event bridge rule %s\n %s", opt.DryRunString(), builder.String())
		} else {
			putRuleInput.State = output.State
		}
	} else {
		putRuleInput.State = eventbridgetypes.RuleStateEnabled
	}
	if opt.DryRun {
		return nil
	}
	putRuleOutput, err := app.aws.PutRule(ctx, putRuleInput)
	if err != nil {
		return err
	}
	log.Printf("[info] update event bridge rule arn `%s`", *putRuleOutput.RuleArn)
	return nil
}

func (app *App) putEventBridgeRuleTargets(ctx context.Context, ruleName, stateMachineArn string, opt DeployOption) error {
	listTargetsOutput, err := app.aws.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
		Rule:  &ruleName,
		Limit: aws.Int32(5),
	})
	if err != nil {
		return err
	}
	putTargetsInput := &eventbridge.PutTargetsInput{
		Rule:    &ruleName,
		Targets: listTargetsOutput.Targets,
	}
	if len(putTargetsInput.Targets) == 0 {
		putTargetsInput.Targets = append(putTargetsInput.Targets, eventbridgetypes.Target{
			Arn: &stateMachineArn,
		})
	}
	for i := range putTargetsInput.Targets {
		if *putTargetsInput.Targets[i].Arn != stateMachineArn {
			continue
		}
		putTargetsInput.Targets[i] = eventbridgetypes.Target{
			Arn:     &stateMachineArn,
			Id:      aws.String(fmt.Sprintf("%s-%s-state-machine", appName, app.cfg.StateMachine.Name)),
			RoleArn: &app.cfg.Schedule.RoleArn,
		}
	}
	if opt.DryRun {
		log.Printf("[notice] update event bridge rule targets %s\n%s", opt.DryRunString(), colorRestString(marshalJSONString(putTargetsInput)))
		return nil
	}
	output, err := app.aws.PutTargets(ctx, putTargetsInput)
	if err != nil {
		return err
	}
	if output.FailedEntryCount != 0 {
		for _, entry := range output.FailedEntries {
			log.Printf("[error] put target failed\n%s", marshalJSONString(entry))
		}
	}
	log.Println("[info] update event bridge rule targes")
	return nil
}
