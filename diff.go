package stefunny

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type DiffOption struct {
	Unified   bool   `name:"unified" help:"output in unified format" short:"u" default:"false" json:"unified,omitempty"`
	Qualifier string `name:"qualifier" help:"qualifier for state machine" default:"" json:"qualifier,omitempty"`
}

func (app *App) Diff(ctx context.Context, opt DiffOption) error {
	newStateMachine := app.cfg.NewStateMachine()
	currentStateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name: app.cfg.StateMachineName(),
	})
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
	}
	ds := strings.TrimSpace(currentStateMachine.DiffString(newStateMachine, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
	}
	var qualified string
	var currentRules EventBridgeRules
	newRules := app.cfg.NewEventBridgeRules()
	if currentStateMachine != nil {
		qualified = currentStateMachine.QualifiedARN(app.StateMachineAliasName())
		currentRules, err = app.eventbridgeSvc.SearchRelatedRules(ctx, &SearchRelatedRulesInput{
			StateMachineQualifiedARN: qualified,
			RuleNames:                newRules.Names(),
		})
		if err != nil {
			return fmt.Errorf("failed to search related rules: %w", err)
		}
	} else {
		qualified = "[known after deploy]:" + app.StateMachineAliasName()
	}
	newRules.SetStateMachineQualifiedARN(qualified)
	ds = strings.TrimSpace(currentRules.DiffString(newRules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
	}
	var currentSchedules Schedules
	newSchedules := app.cfg.NewSchedules()
	if currentStateMachine != nil {
		currentSchedules, err = app.schedulerSvc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
			StateMachineQualifiedARN: qualified,
			ScheduleNames:            newSchedules.Names(),
		})
		if err != nil {
			return fmt.Errorf("failed to search related schedules: %w", err)
		}
	}
	newSchedules.SetStateMachineQualifiedARN(qualified)
	ds = strings.TrimSpace(currentSchedules.DiffString(newSchedules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
	}
	return nil
}
