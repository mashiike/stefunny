package stefunny

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func (app *App) Create(ctx context.Context, opt CreateOption) error {
	log.Println("[info] Starting create", opt.DryRunString())
	if err := app.createStateMachine(ctx, opt); err != nil {
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
				Key:   aws.String("ManagedBy"),
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
	output, err := app.sfn.CreateStateMachine(ctx, input)
	if err != nil {
		return fmt.Errorf("create failed: %w", err)
	}

	log.Printf("[notice] created arn `%s`", *output.StateMachineArn)
	return nil
}
