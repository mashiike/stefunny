package stefunny

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

func (app *App) Create(ctx context.Context, opt DeployOption) error {
	log.Println("[info] Starting create", opt.DryRunString())
	err := app.createStateMachine(ctx, opt)
	if err != nil {
		return err
	}
	if err := app.putSchedule(ctx, opt); err != nil {
		return err
	}
	log.Println("[info] finish create", opt.DryRunString())
	return nil
}

func (app *App) createStateMachine(ctx context.Context, opt DeployOption) error {
	definition, err := app.cfg.LoadDefinition()
	if err != nil {
		return fmt.Errorf("load definition failed: %w", err)
	}
	logging, err := app.LoadLoggingConfiguration(ctx)
	if err != nil {
		return fmt.Errorf("load logging config failed: %w", err)
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
	}
	if opt.DryRun {
		log.Printf("[notice] create state machine %s\n%s", opt.DryRunString(), stateMachine.String())
		return nil
	}
	output, err := app.aws.DeployStateMachine(ctx, stateMachine)
	if err != nil {
		return fmt.Errorf("create failed: %w", err)
	}

	log.Printf("[notice] created arn `%s`", *output.StateMachineArn)
	return nil
}
