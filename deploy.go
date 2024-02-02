package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/mashiike/stefunny/internal/jsonutil"
)

type DeployCommandOption struct {
	DryRun bool `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
}

type ScheduleCommandOption struct {
	DryRun   bool `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	Enabled  bool `name:"enabled" help:"Enable schedule" xor:"schedule" required:"" json:"enabled,omitempty"`
	Disabled bool `name:"disabled" help:"Disable schedule" xor:"schedule" required:"" json:"disabled,omitempty"`
}

type DeployOption struct {
	DryRun                 bool
	ScheduleEnabled        *bool
	SkipDeployStateMachine bool
}

func (opt DeployOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

func (app *App) Deploy(ctx context.Context, opt DeployOption) error {
	log.Println("[info] Starting deploy", opt.DryRunString())
	if !opt.SkipDeployStateMachine {
		if err := app.deployStateMachine(ctx, opt); err != nil {
			return err
		}
	}
	if err := app.deployScheduleRule(ctx, opt); err != nil {
		return err
	}
	log.Println("[info] finish deploy", opt.DryRunString())
	return nil
}

func (app *App) deployStateMachine(ctx context.Context, opt DeployOption) error {
	stateMachine, err := app.aws.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		if err == ErrStateMachineDoesNotExist && !opt.SkipDeployStateMachine {
			return app.createStateMachine(ctx, opt)
		}
		return fmt.Errorf("failed to describe current state machine status: %w", err)
	}
	newStateMachine, err := app.LoadStateMachine(ctx)
	if err != nil {
		return err
	}
	newStateMachine.StateMachineArn = stateMachine.StateMachineArn
	if opt.DryRun {
		diffString := stateMachine.DiffString(newStateMachine)
		log.Printf("[notice] change state machine %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	output, err := app.aws.DeployStateMachine(ctx, newStateMachine)
	if err != nil {
		return err
	}
	log.Printf("[info] deploy state machine `%s`(at `%s`)\n", app.cfg.StateMachine.Name, *output.UpdateDate)
	return nil
}

func (app *App) deployScheduleRule(ctx context.Context, opt DeployOption) error {
	stateMachineArn, err := app.aws.GetStateMachineArn(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return fmt.Errorf("failed to get state machine arn: %w", err)
	}
	rules, err := app.aws.SearchScheduleRule(ctx, stateMachineArn)
	if err != nil {
		return err
	}
	newRules, err := app.LoadScheduleRules(ctx, stateMachineArn)
	if err != nil {
		return err
	}
	newRules.SetStateMachineArn(stateMachineArn)
	if opt.ScheduleEnabled != nil {
		newRules.SetEnabled(*opt.ScheduleEnabled)
	} else {
		newRules.SyncState(rules)
	}

	//Ignore no managed rule
	noConfigRules := rules.Subtract(newRules)
	noManageRules := make(ScheduleRules, 0, len(noConfigRules))
	for _, rule := range noConfigRules {
		if !rule.HasTagKeyValue(tagManagedBy, appName) {
			log.Printf("[warn] found a scheduled rule `%s` that %s does not manage.", *rule.Name, appName)
			noManageRules = append(noManageRules, rule)
		}
	}
	rules = rules.Exclude(noManageRules)

	if len(rules) == 0 && len(newRules) == 0 {
		log.Println("[debug] no thing to do")
		return nil
	}
	if len(rules) == 0 && len(newRules) != 0 {
		return app.createScheduleRule(ctx, opt)
	}

	deleteRules := rules.Exclude(newRules)
	log.Printf("[debug] delete rules:\n%s\n", jsonutil.MarshalJSONString(deleteRules))
	if opt.DryRun {
		diffString := rules.DiffString(newRules)
		log.Printf("[notice] change schedule rule %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	if len(deleteRules) != 0 {
		log.Printf("[debug] try delete %d rules", len(deleteRules))
		err := app.aws.DeleteScheduleRules(ctx, deleteRules)
		if err != nil {
			return err
		}
		log.Printf("[info] delete %d schedule rule", len(deleteRules))
		return nil
	}
	output, err := app.aws.DeployScheduleRules(ctx, newRules)
	if err != nil {
		return err
	}
	if output.FailedEntryCount() != 0 {
		for _, o := range output {
			log.Printf("[error] deploy schedule rule with failed entries %s", jsonutil.MarshalJSONString(o.FailedEntries))
		}
		return errors.New("failed entry count > 0")
	}
	for _, o := range output {
		log.Printf("[info] deploy schedule rule %s", *o.RuleArn)
	}
	return nil
}
