package stefunny

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// DiffOption configures App.Diff.
type DiffOption struct {
	Unified   bool   `name:"unified" help:"output in unified format" short:"u" default:"true" negatable:"" json:"unified,omitempty"`
	Qualifier string `name:"qualifier" help:"qualifier for state machine" default:"" json:"qualifier,omitempty"`
	ExitCode  bool   `name:"exit-code" help:"exit with code 2 if there are differences" default:"false" json:"exit_code,omitempty"`
}

// ErrHasDiff is returned by App.Diff when DiffOption.ExitCode is set and a
// difference was found, so the caller can map it to a distinct exit code.
var ErrHasDiff = errors.New("there are differences")

// Diff prints the diff between the config and the deployed state machine,
// EventBridge rules and EventBridge Scheduler schedules. Returns ErrHasDiff
// if opt.ExitCode is set and a difference was found.
func (app *App) Diff(ctx context.Context, opt DiffOption) error {
	sfnSvc, err := app.sfnService(ctx)
	if err != nil {
		return err
	}
	newStateMachine := app.cfg.NewStateMachine()
	var stateMachineArn string
	currentStateMachine, err := sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name:      app.cfg.StateMachineName(),
		Qualifier: opt.Qualifier,
	})
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
		if opt.Qualifier != "" {
			latestStateMachine, err := sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
				Name: app.cfg.StateMachineName(),
			})
			if err != nil {
				if !errors.Is(err, ErrStateMachineDoesNotExist) {
					return fmt.Errorf("failed to describe latest state machine status: %w", err)
				}
			}
			stateMachineArn = latestStateMachine.QualifiedArn(app.StateMachineAliasName())
		}
	} else {
		stateMachineArn = currentStateMachine.QualifiedArn(app.StateMachineAliasName())
	}
	if stateMachineArn == "" {
		stateMachineArn = "[known after deploy]:" + app.StateMachineAliasName()
	}
	newStateMachine.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	hasDiff := false
	ds := strings.TrimSpace(currentStateMachine.DiffString(newStateMachine, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
		hasDiff = true
	}
	var currentRules EventBridgeRules
	newRules := app.cfg.NewEventBridgeRules()
	if currentStateMachine != nil {
		eventBridgeSvc, err := app.eventBridgeService(ctx)
		if err != nil {
			return err
		}
		currentRules, err = eventBridgeSvc.SearchRelatedRules(ctx, &SearchRelatedRulesInput{
			StateMachineQualifiedArn: stateMachineArn,
			RuleNames:                newRules.Names(),
		})
		if err != nil {
			return fmt.Errorf("failed to search related rules: %w", err)
		}
	}
	newRules.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	newRules.SetStateMachineQualifiedArn(stateMachineArn)
	newRules.SyncState(currentRules)
	ds = strings.TrimSpace(currentRules.DiffString(newRules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
		hasDiff = true
	}
	var currentSchedules Schedules
	newSchedules := app.cfg.NewSchedules()
	if currentStateMachine != nil {
		schedulerSvc, err := app.schedulerService(ctx)
		if err != nil {
			return err
		}
		currentSchedules, err = schedulerSvc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
			StateMachineQualifiedArn: stateMachineArn,
			ScheduleNames:            newSchedules.Names(),
		})
		if err != nil {
			return fmt.Errorf("failed to search related schedules: %w", err)
		}
	}
	newSchedules.SetStateMachineQualifiedArn(stateMachineArn)
	newSchedules.SyncState(currentSchedules)
	ds = strings.TrimSpace(currentSchedules.DiffString(newSchedules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
		hasDiff = true
	}
	if opt.ExitCode && hasDiff {
		return ErrHasDiff
	}
	return nil
}
