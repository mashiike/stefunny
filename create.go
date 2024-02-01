package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/mashiike/stefunny/internal/jsonutil"
)

func (app *App) Create(ctx context.Context, opt DeployOption) error {
	log.Println("[warn] create command is deprecated. use deploy command instead")
	log.Println("[info] Starting create", opt.DryRunString())
	err := app.createStateMachine(ctx, opt)
	if err != nil {
		return err
	}
	if err := app.createScheduleRule(ctx, opt); err != nil {
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

func (app *App) createScheduleRule(ctx context.Context, opt DeployOption) error {
	if app.cfg.Schedule == nil {
		log.Println("[debug] schedule rule is not set")
		return nil
	}
	if opt.DryRun {
		rules, err := app.LoadScheduleRules(ctx, "[state machine arn]")
		if err != nil {
			return err
		}
		log.Printf("[notice] create schedule rules %s\n%s", opt.DryRunString(), rules.String())
		return nil
	}
	stateMachineArn, err := app.aws.GetStateMachineArn(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return fmt.Errorf("failed to get state machine arn: %w", err)
	}
	rules, err := app.LoadScheduleRules(ctx, stateMachineArn)
	if err != nil {
		return err
	}
	output, err := app.aws.DeployScheduleRules(ctx, rules)
	if err != nil {
		return err
	}
	log.Printf("[info] deploy schedule rule %s\n", jsonutil.MarshalJSONString(output))
	if output.FailedEntryCount() != 0 {
		return errors.New("failed entry count > 0")
	}
	return nil
}
