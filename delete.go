package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
)

func (app *App) Delete(ctx context.Context, opt DeleteOption) error {
	log.Println("[info] Starting delete", opt.DryRunString())
	stateMachine, err := app.aws.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}

	log.Printf("[notice] delete state machine is %s\n%s", opt.DryRunString(), stateMachine)

	rules, err := app.aws.SearchScheduleRule(ctx, *stateMachine.StateMachineArn)
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
		name, err := prompt(ctx, `Enter the state machine name to DELETE`, "")
		if err != nil {
			return err
		}
		if !strings.EqualFold(name, app.cfg.StateMachine.Name) {
			log.Println("[info] Aborted")
			return errors.New("confirmation failed")
		}
	}
	err = app.aws.DeleteStateMachine(ctx, stateMachine)
	if err != nil {
		return fmt.Errorf("failed to delete state machine status: %w", err)
	}
	if len(rules) > 0 {
		err := app.aws.DeleteScheduleRules(ctx, rules)
		if err != nil {
			return fmt.Errorf("failed to delete rules: %w", err)
		}
	}
	log.Println("[info] finish delete", opt.DryRunString())
	return nil
}
