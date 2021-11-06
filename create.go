package stefunny

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func (app *App) Create(ctx context.Context, opt CreateOption) error {
	log.Println("[info] Starting create", opt.DryRunString())
	err := app.createStateMachine(ctx, opt)
	if err != nil {
		return err
	}
	if err := app.createSchedule(ctx, opt); err != nil {
		return err
	}
	log.Println("[info] finish create", opt.DryRunString())
	return nil
}

func (app *App) createStateMachine(ctx context.Context, opt CreateOption) error {
	definition, err := app.cfg.LoadDefinition()
	if err != nil {
		return fmt.Errorf("load definition failed: %w", err)
	}
	logging, err := app.LoadLoggingConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("load logging config failed: %w", err)
	}
	input := &sfn.CreateStateMachineInput{
		Name:                 &app.cfg.StateMachine.Name,
		Type:                 app.cfg.StateMachine.stateMachineType,
		RoleArn:              &app.cfg.StateMachine.RoleArn,
		LoggingConfiguration: logging,
		TracingConfiguration: app.cfg.StateMachine.LoadTracingConfiguration(),
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	}
	if opt.DryRun {
		log.Printf("[notice] create parameters %s\n%s", opt.DryRunString(), colorRestString(marshalJSONString(input)))
		log.Printf("[notice] create state machine defeinition %s\n%s", opt.DryRunString(), colorRestString(definition))
		return nil
	}

	input.Definition = &definition
	output, err := app.aws.CreateStateMachine(ctx, input)
	if err != nil {
		return fmt.Errorf("create failed: %w", err)
	}

	log.Printf("[notice] created arn `%s`", *output.StateMachineArn)
	return nil
}

func (app *App) createSchedule(ctx context.Context, opt CreateOption) error {
	if app.cfg.Schedule == nil {
		log.Println("[debug] schedule is not set")
		return nil
	}
	stateMachineArn, err := app.aws.GetStateMachineArn(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	output, err := app.aws.PutRule(ctx, &eventbridge.PutRuleInput{
		Name:               aws.String(getScheduleRuleName(app.cfg.StateMachine.Name)),
		Description:        aws.String(fmt.Sprintf("for state machine %s schedule", stateMachineArn)),
		ScheduleExpression: &app.cfg.Schedule.Expression,
		State:              eventbridgetypes.RuleStateEnabled,
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	})
	if err != nil {
		return err
	}
	log.Printf("[notice] created rule arn `%s`", *output.RuleArn)
	return nil
}
