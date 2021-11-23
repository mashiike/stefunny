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

	rules := make(ScheduleRules, 0, len(app.cfg.Schedule))
	for _, cfg := range app.cfg.Schedule {
		rule, err := app.aws.DescribeScheduleRule(ctx, cfg.RuleName)
		if err == nil {
			log.Printf("[notice] delete schedule rule is %s\n%s", opt.DryRunString(), rule)
		} else if err != nil && err != ErrScheduleRuleDoesNotExist {
			return err
		}
		rules = append(rules, rule)
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
