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
	SkipTrigger            bool   `name:"skip-trigger" help:"Skip deploy trigger" json:"skip_trigger,omitempty"`
	VersionDescription     string `name:"version-description" help:"Version description" json:"version_description,omitempty"`
	KeepVersions           int    `help:"Number of latest versions to keep. Older versions will be deleted. (Optional value: default 0)" default:"0" json:"keep_versions,omitempty"`
	AliasName              string `name:"alias" help:"alias name for publish" default:"current" json:"alias,omitempty"`
}

func (cmd *DeployCommandOption) DeployOption() DeployOption {
	return DeployOption{
		DryRun:                 cmd.DryRun,
		SkipDeployStateMachine: cmd.SkipDeployStateMachine,
		SkipTrigger:            cmd.SkipTrigger,
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
		SkipTrigger:            false,
		SkipDeployStateMachine: true,
		AliasName:              cmd.AliasName,
	}
}

type DeployOption struct {
	DryRun                 bool
	ScheduleEnabled        *bool
	SkipDeployStateMachine bool
	SkipTrigger            bool
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
			return fmt.Errorf("failed to deploy state machine: %w", err)
		}
	}
	if !opt.SkipTrigger {
		if err := app.deployEventBridgeRules(ctx, opt); err != nil {
			return fmt.Errorf("failed to deploy event bridge rules: %w", err)
		}
	}

	log.Println("[info] finish deploy", opt.DryRunString())
	return nil
}

func (app *App) deployStateMachine(ctx context.Context, opt DeployOption) error {
	newStateMachine := app.cfg.NewStateMachine()
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

func (app *App) deployEventBridgeRules(ctx context.Context, opt DeployOption) error {
	stateMachineARN, err := app.sfnSvc.GetStateMachineArn(ctx, app.cfg.StateMachineName())
	if err != nil {
		return fmt.Errorf("failed to get state machine arn: %w", err)
	}
	newRules := app.cfg.NewEventBridgeRules()
	targetARN := qualifiedARN(stateMachineARN, opt.AliasName)
	newRules.SetStateMachineQualifiedARN(targetARN)
	keepState := true
	if opt.ScheduleEnabled != nil {
		newRules.SetEnabled(*opt.ScheduleEnabled)
		keepState = false
	}
	if opt.DryRun {
		currentRules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, targetARN)
		if err != nil {
			return fmt.Errorf("failed to search related rules: %w", err)
		}
		if keepState {
			newRules.SyncState(currentRules)
		}
		diffString := currentRules.DiffString(newRules)
		log.Printf("[notice] change related rules %s\n%s", opt.DryRunString(), diffString)
		return nil
	}
	if err := app.eventbridgeSvc.DeployRules(ctx, targetARN, newRules, keepState); err != nil {
		return fmt.Errorf("failed to deploy rules: %w", err)
	}
	return nil
}
