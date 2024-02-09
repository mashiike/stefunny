package stefunny

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type DiffOption struct {
	Unified   bool   `name:"unified" help:"output in unified format" short:"u" default:"false" json:"unified,omitempty"`
	AliasName string `name:"alias" help:"alias name" default:"current" json:"alias,omitempty"`
}

func (app *App) Diff(ctx context.Context, opt DiffOption) error {
	newStateMachine := app.cfg.NewStateMachine()
	currentStateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
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
	if currentStateMachine != nil {
		qualified = currentStateMachine.QualifiedARN(opt.AliasName)
		currentRules, err = app.eventbridgeSvc.SearchRelatedRules(ctx, qualified)
		if err != nil {
			return fmt.Errorf("failed to search related rules: %w", err)
		}
	}
	newRules := app.cfg.NewEventBridgeRules()
	newRules.SetStateMachineQualifiedARN(qualified)
	ds = strings.TrimSpace(currentRules.DiffString(newRules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
	}
	var currentSchedules Schedules
	if currentStateMachine != nil {
		currentSchedules, err = app.schedulerSvc.SearchRelatedSchedules(ctx, qualified)
		if err != nil {
			return fmt.Errorf("failed to search related schedules: %w", err)
		}
	}
	newSchedules := app.cfg.NewSchedules()
	newSchedules.SetStateMachineQualifiedARN(qualified)
	ds = strings.TrimSpace(currentSchedules.DiffString(newSchedules, opt.Unified))
	if ds != "" {
		fmt.Println(ds)
	}
	return nil
}
