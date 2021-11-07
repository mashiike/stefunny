package stefunny

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
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
		if _, ok := err.(*sfntypes.StateMachineDoesNotExist); ok {
			return app.createStateMachine(ctx, opt)
		}
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}
	newStateMachine, err := app.LoadStateMachine(ctx)
	if err != nil {
		return err
	}
	newStateMachine.StateMachineArn = stateMachine.StateMachineArn
	if opt.DryRun {
		diffString := stateMachine.DiffString(newStateMachine)
		log.Printf("[notice] change state machine %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	output, err := app.aws.DeployStateMachine(ctx, newStateMachine)
	if err != nil {
		return err
	}
	log.Printf("[info] deploy state machine `%s`(at `%s`)\n", app.cfg.StateMachine.Name, *output.UpdateDate)
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
