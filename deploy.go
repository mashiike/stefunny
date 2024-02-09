package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type DeployCommandOption struct {
	DryRun             bool   `name:"dry-run" help:"Dry run" json:"dry_run,omitempty"`
	SkipStateMachine   bool   `name:"skip-state-machine" help:"Skip deploy state machine" json:"skip_state_machine,omitempty"`
	SkipTrigger        bool   `name:"skip-trigger" help:"Skip deploy trigger" json:"skip_trigger,omitempty"`
	VersionDescription string `name:"version-description" help:"Version description" json:"version_description,omitempty"`
	KeepVersions       int    `help:"Number of latest versions to keep. Older versions will be deleted. (Optional value: default 0)" default:"0" json:"keep_versions,omitempty"`
	AliasName          string `name:"alias" help:"alias name for publish" default:"current" json:"alias,omitempty"`
	TriggerEnabled     bool   `name:"trigger-enabled" help:"Enable trigger" xor:"trigger" json:"trigger_enabled,omitempty"`
	TriggerDisabled    bool   `name:"trigger-disabled" help:"Disable trigger" xor:"trigger" json:"trigger_disabled,omitempty"`
	Unified            bool   `name:"unified" help:"when dry run, output unified diff" negative:"" default:"true" json:"unified,omitempty"`
}

func (cmd *DeployCommandOption) DeployOption() DeployOption {
	var enabled *bool
	if cmd.TriggerEnabled {
		enabled = ptr(true)
	}
	if cmd.TriggerDisabled {
		enabled = ptr(false)
	}
	return DeployOption{
		DryRun:             cmd.DryRun,
		SkipStateMachine:   cmd.SkipStateMachine,
		SkipTrigger:        cmd.SkipTrigger,
		VersionDescription: cmd.VersionDescription,
		KeepVersions:       cmd.KeepVersions,
		AliasName:          cmd.AliasName,
		ScheduleEnabled:    enabled,
		Unified:            cmd.Unified,
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
		DryRun:           cmd.DryRun,
		ScheduleEnabled:  enabled,
		SkipTrigger:      false,
		SkipStateMachine: true,
		AliasName:        cmd.AliasName,
	}
}

type DeployOption struct {
	DryRun             bool
	ScheduleEnabled    *bool
	SkipStateMachine   bool
	SkipTrigger        bool
	VersionDescription string
	KeepVersions       int
	AliasName          string
	Unified            bool
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
	if !opt.SkipStateMachine {
		if err := app.deployStateMachine(ctx, opt); err != nil {
			return fmt.Errorf("failed to deploy state machine: %w", err)
		}
	}
	if !opt.SkipTrigger {
		if err := app.deployEventBridgeRules(ctx, opt); err != nil {
			return fmt.Errorf("failed to deploy event bridge rules: %w", err)
		}
		if err := app.deploySchedules(ctx, opt); err != nil {
			return fmt.Errorf("failed to deploy schedules: %w", err)
		}
	}
	log.Println("[info] finish deploy", opt.DryRunString())
	return nil
}

func (app *App) deployStateMachine(ctx context.Context, opt DeployOption) error {
	log.Println("[debug] deploy state machine")
	newStateMachine := app.cfg.NewStateMachine()
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
	if err != nil {
		log.Printf("[debug] describe state machine error %#v", err)
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
	} else {
		newStateMachine.StateMachineArn = stateMachine.StateMachineArn
	}
	if opt.DryRun {
		diffString := stateMachine.DiffString(newStateMachine, opt.Unified)
		log.Printf("[notice] change state machine %s\n", opt.DryRunString())
		fmt.Println(diffString)
		return nil
	}
	if opt.VersionDescription != "" {
		newStateMachine.VersionDescription = aws.String(opt.VersionDescription)
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
	isStateMachineFound := true
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to get state machine arn: %w", err)
		}
		stateMachineARN = "[known after deploy]"
		isStateMachineFound = false
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
		currentRules := EventBridgeRules{}
		if isStateMachineFound {
			currentRules, err = app.eventbridgeSvc.SearchRelatedRules(ctx, targetARN)
			if err != nil {
				return fmt.Errorf("failed to search related rules: %w", err)
			}
		}
		sameNameRules, err := app.eventbridgeSvc.SearchRulesByNames(ctx, newRules.Names(), targetARN)
		if err != nil {
			return fmt.Errorf("failed to search related rules: %w", err)
		}
		currentRules = currentRules.Merge(sameNameRules)
		if keepState {
			newRules.SyncState(currentRules)
		}
		diffString := currentRules.DiffString(newRules, opt.Unified)
		log.Printf("[notice] change related rules %s\n", opt.DryRunString())
		fmt.Println(diffString)
		return nil
	}
	if err := app.eventbridgeSvc.DeployRules(ctx, targetARN, newRules, keepState); err != nil {
		return fmt.Errorf("failed to deploy rules: %w", err)
	}
	return nil
}

func (app *App) deploySchedules(ctx context.Context, opt DeployOption) error {
	stateMachineARN, err := app.sfnSvc.GetStateMachineArn(ctx, app.cfg.StateMachineName())
	isStateMachineFound := true
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {

			return fmt.Errorf("failed to get state machine arn: %w", err)
		}
		stateMachineARN = "[known after deploy]"
		isStateMachineFound = false
	}
	newSchedules := app.cfg.NewSchedules()
	targetARN := qualifiedARN(stateMachineARN, opt.AliasName)
	newSchedules.SetStateMachineQualifiedARN(targetARN)
	if opt.DryRun {
		currentSchedules := Schedules{}
		if isStateMachineFound {
			currentSchedules, err = app.schedulerSvc.SearchRelatedSchedules(ctx, targetARN)
			if err != nil {
				return fmt.Errorf("failed to search related schedules: %w", err)
			}
		}
		diffString := currentSchedules.DiffString(newSchedules, opt.Unified)
		log.Printf("[notice] change related schedules %s", opt.DryRunString())
		fmt.Println(diffString)
		return nil
	}
	if err := app.schedulerSvc.DeploySchedules(ctx, targetARN, newSchedules, opt.KeepVersions > 0); err != nil {
		return fmt.Errorf("failed to deploy schedules: %w", err)
	}
	return nil
}
