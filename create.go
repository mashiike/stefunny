package stefunny

import (
	"context"
	"fmt"
	"log"
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
	stateMachine, err := app.LoadStateMachine(ctx)
	if err != nil {
		return err
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
