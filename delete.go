package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

type DeleteOption struct {
	DryRun bool `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	Force  bool `name:"force" help:"delete without confirmation" json:"force,omitempty"`
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

	rules, err := app.eventbridgeSvc.SearchScheduleRule(ctx, *stateMachine.StateMachineArn)
	if err != nil {
		return err
	}
	//Ignore no managed rule
	noManageRules := make(ScheduleRules, 0, len(rules))
	for _, rule := range rules {
		if !rule.HasTagKeyValue(tagManagedBy, appName) {
			log.Printf("[warn] found a scheduled rule `%s` that %s does not manage. this rule is not delete.", *rule.Name, appName)
			noManageRules = append(noManageRules, rule)
		}
	}
	rules = rules.Exclude(noManageRules)
	for _, rule := range rules {
		log.Printf("[notice] delete schedule rule is %s\n%s", opt.DryRunString(), rule)
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
	if len(rules) > 0 {
		err := app.eventbridgeSvc.DeleteScheduleRules(ctx, rules)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %w", err)
		}
	}
	log.Println("[info] finish delete", opt.DryRunString())
	return nil
}
