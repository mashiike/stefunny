package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type DeployCommandOption struct {
	DryRun                 bool   `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	SkipDeployStateMachine bool   `name:"skip-deploy-state-machine" help:"Skip deploy state machine" json:"skip_deploy_state_machine,omitempty"`
	VersionDescription     string `name:"version-description" help:"Version description" json:"version_description,omitempty"`
	KeepVersions           int    `help:"Number of latest versions to keep. Older versions will be deleted. (Optional value: default 0)" default:"0" json:"keep_versions,omitempty"`
	AliasName              string `name:"alias" help:"alias name for publish" default:"current" json:"alias,omitempty"`
}

func (cmd *DeployCommandOption) DeployOption() DeployOption {
	return DeployOption{
		DryRun:                 cmd.DryRun,
		SkipDeployStateMachine: cmd.SkipDeployStateMachine,
		VersionDescription:     cmd.VersionDescription,
		KeepVersions:           cmd.KeepVersions,
		AliasName:              cmd.AliasName,
	}
}

type ScheduleCommandOption struct {
	DryRun    bool   `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	Enabled   bool   `name:"enabled" help:"Enable schedule" xor:"schedule" required:"" json:"enabled,omitempty"`
	Disabled  bool   `name:"disabled" help:"Disable schedule" xor:"schedule" required:"" json:"disabled,omitempty"`
	AliasName string `name:"alias" help:"alias name for publish" default:"current" json:"alias,omitempty"`
}

func (cmd *ScheduleCommandOption) DeployOption() DeployOption {
	var enabled *bool
	if cmd.Enabled {
		enabled = ptr(true)
	}
	if cmd.Disabled {
		enabled = ptr(false)
	}
	return DeployOption{
		DryRun:                 cmd.DryRun,
		ScheduleEnabled:        enabled,
		SkipDeployStateMachine: true,
		AliasName:              cmd.AliasName,
	}
}

type DeployOption struct {
	DryRun                 bool
	ScheduleEnabled        *bool
	SkipDeployStateMachine bool
	VersionDescription     string
	KeepVersions           int
	AliasName              string
}

func (opt DeployOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

func (app *App) Deploy(ctx context.Context, opt DeployOption) error {
	log.Println("[info] Starting deploy", opt.DryRunString())
	if opt.AliasName != "" {
		app.sfnSvc.SetAliasName(opt.AliasName)
	}
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
	newStateMachine, err := app.LoadStateMachine()
	if err != nil {
		return err
	}
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
	} else {
		newStateMachine.StateMachineArn = stateMachine.StateMachineArn
	}
	if opt.DryRun {
		diffString := stateMachine.DiffString(newStateMachine)
		log.Printf("[notice] change state machine %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	if opt.VersionDescription != "" {
		newStateMachine.CreateStateMachineInput.VersionDescription = aws.String(opt.VersionDescription)
	}
	output, err := app.sfnSvc.DeployStateMachine(ctx, newStateMachine)
	if err != nil {
		return err
	}
	log.Printf("[info] deploy state machine `%s`(at `%s`)\n", app.cfg.StateMachineName(), *output.UpdateDate)
	if opt.KeepVersions > 0 {
		if err := app.sfnSvc.PurgeStateMachineVersions(ctx, newStateMachine, opt.KeepVersions); err != nil {
			return fmt.Errorf("failed to delete older versions: %w", err)
		}
	}
	return nil
}

func (app *App) deployScheduleRule(ctx context.Context, opt DeployOption) error {
	stateMachineArn, err := app.sfnSvc.GetStateMachineArn(ctx, app.cfg.StateMachineName())
	if err != nil {
		return fmt.Errorf("failed to get state machine arn: %w", err)
	}
	rules, err := app.eventbridgeSvc.SearchScheduleRule(ctx, stateMachineArn)
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
		if !rule.IsManagedBy() {
			log.Printf("[warn] found a scheduled rule `%s` that %s does not manage.", *rule.Name, appName)
			noManageRules = append(noManageRules, rule)
		}
	}
	rules = rules.Exclude(noManageRules)

	if len(rules) == 0 && len(newRules) == 0 {
		log.Println("[debug] no thing to do")
		return nil
	}

	deleteRules := rules.Exclude(newRules)
	log.Printf("[debug] delete rules:\n%s\n", MarshalJSONString(deleteRules))
	if opt.DryRun {
		diffString := rules.DiffString(newRules)
		log.Printf("[notice] change schedule rule %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	if len(deleteRules) != 0 {
		log.Printf("[debug] try delete %d rules", len(deleteRules))
		err := app.eventbridgeSvc.DeleteScheduleRules(ctx, deleteRules)
		if err != nil {
			return err
		}
		log.Printf("[info] delete %d schedule rule", len(deleteRules))
		return nil
	}
	output, err := app.eventbridgeSvc.DeployScheduleRules(ctx, newRules)
	if err != nil {
		return err
	}
	if output.FailedEntryCount() != 0 {
		for _, o := range output {
			log.Printf("[error] deploy schedule rule with failed entries %s", MarshalJSONString(o.FailedEntries))
		}
		return errors.New("failed entry count > 0")
	}
	for _, o := range output {
		log.Printf("[info] deploy schedule rule %s", *o.RuleArn)
	}
	return nil
}
