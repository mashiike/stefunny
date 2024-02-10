package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
)

type RollbackOption struct {
	DryRun      bool `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	KeepVersion bool `name:"keep-version" help:"Keep current version, no delete" json:"keep_version,omitempty"`
}

func (opt RollbackOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

func (app *App) Rollback(ctx context.Context, opt RollbackOption) error {
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
	if err != nil {
		if errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("state machine `%s` is not found", app.cfg.StateMachineName())
		}
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}

	log.Println("[info] Starting rollback", coalesce(stateMachine.StateMachineArn), opt.DryRunString())
	if err := app.sfnSvc.RollbackStateMachine(ctx, stateMachine, opt.KeepVersion, opt.DryRun); err != nil {
		return err
	}
	log.Println("[info] finish rollback", coalesce(stateMachine.StateMachineArn), opt.DryRunString())
	return nil
}
