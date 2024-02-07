package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

type DeleteOption struct {
	DryRun    bool   `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	Force     bool   `name:"force" help:"delete without confirmation" json:"force,omitempty"`
	AliasName string `name:"alias" help:"alias name" default:"current" json:"alias,omitempty"`
}

func (opt DeleteOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

func (app *App) Delete(ctx context.Context, opt DeleteOption) error {
	log.Println("[info] Starting delete", opt.DryRunString())
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
	if err != nil {
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}

	log.Printf("[notice] delete state machine is %s\n%s", opt.DryRunString(), stateMachine)
	currentRules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, stateMachine.QualifiedARN(opt.AliasName))
	if err != nil {
		return fmt.Errorf("failed to search related rules: %w", err)
	}
	if len(currentRules) > 0 {
		log.Printf("[notice] delete related rules is %s\n%s", opt.DryRunString(), currentRules)
	}
	if opt.DryRun {
		log.Println("[info] dry run ok")
		return nil
	}
	if !opt.Force {
		name, err := prompt(ctx, fmt.Sprintf(`Enter the state machine name [%s] to DELETE`, app.cfg.StateMachineName()), "")
		if err != nil {
			return err
		}
		if !strings.EqualFold(name, app.cfg.StateMachineName()) {
			log.Println("[info] Aborted")
			return errors.New("confirmation failed")
		}
	}
	err = app.sfnSvc.DeleteStateMachine(ctx, stateMachine)
	if err != nil {
		return fmt.Errorf("failed to delete state machine status: %w", err)
	}
	if len(currentRules) > 0 {
		err := app.eventbridgeSvc.DeployRules(ctx, stateMachine.QualifiedARN(opt.AliasName), EventBridgeRules{}, false)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %w", err)
		}
	}
	log.Println("[info] finish delete", opt.DryRunString())
	return nil
}
