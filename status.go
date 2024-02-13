package stefunny

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type StatusOption struct {
	Format string `name:"format" help:"output format(text,json)" default:"text" enum:"text,json" json:"format,omitempty"`
	Latest bool   `name:"latest" help:"show latest state machine" default:"false" json:"latest,omitempty"`
}

func (app *App) Status(ctx context.Context, opt StatusOption) error {
	quarifier := app.StateMachineAliasName()
	if opt.Latest {
		quarifier = ""
	}
	stateMachineStatus, err := app.newStateMachineStatus(ctx, quarifier)
	if err != nil {
		return fmt.Errorf("failed to get state machine status: %w", err)
	}
	rulesStatus, err := app.newRuleStatus(ctx, stateMachineStatus.Arn)
	if err != nil {
		return fmt.Errorf("failed to get rule status: %w", err)
	}
	scheduleStatus, err := app.newScheduleStatus(ctx, stateMachineStatus.Arn)
	if err != nil {
		return fmt.Errorf("failed to get schedule status: %w", err)
	}
	status := &StatusOutput{
		StateMachine:         stateMachineStatus,
		EventBridge:          rulesStatus,
		EventBridgeScheduler: scheduleStatus,
	}
	switch opt.Format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(status); err != nil {
			return fmt.Errorf("failed to encode status: %w", err)
		}
		return nil
	default:
		fmt.Fprintln(os.Stdout, status)
		return nil
	}
}

func (app *App) newStateMachineStatus(ctx context.Context, qualifier string) (*StateMachineStatus, error) {
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name:      app.cfg.StateMachineName(),
		Qualifier: qualifier,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to describe state machine: %w", err)
	}
	version, err := extructVersion(coalesce(stateMachine.StateMachineArn))
	if err != nil {
		version = 0
	}
	stateMachineArn := removeQualifierFromArn(coalesce(stateMachine.StateMachineArn))
	return &StateMachineStatus{
		Arn:            stateMachineArn,
		Name:           app.cfg.StateMachineName(),
		CurrentVersion: version,
		Status:         string(stateMachine.Status),
		CreatedAt:      (coalesce(stateMachine.CreationDate)).Format(time.RFC3339),
	}, nil
}

func (app *App) newRuleStatus(ctx context.Context, stateMachineArn string) ([]*RulesStatus, error) {
	cfgRules := app.cfg.NewEventBridgeRules()
	stateMachineQualifiedArn := addQualifierToArn(stateMachineArn, app.StateMachineAliasName())
	rules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, &SearchRelatedRulesInput{
		StateMachineQualifiedArn: stateMachineQualifiedArn,
		RuleNames:                cfgRules.Names(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search related rules: %w", err)
	}
	rulesStatus := make([]*RulesStatus, 0, len(rules))
	for _, rule := range rules {
		status := &RulesStatus{
			RuleArn:  coalesce(rule.RuleArn),
			RuleName: coalesce(rule.Name),
			Status:   string(rule.State),
		}
		if rule.ScheduleExpression != nil {
			status.ScheduleExpression = rule.ScheduleExpression
		}
		if rule.EventPattern != nil {
			status.EventPattern = rule.EventPattern
		}
		targetQuarifier := strings.TrimPrefix(coalesce(rule.Target.Arn), stateMachineArn)
		if targetQuarifier == "" {
			targetQuarifier = "$LATEST"
		}
		status.Target = strings.TrimLeft(targetQuarifier, ":")
		rulesStatus = append(rulesStatus, status)
	}
	for _, cfgRule := range cfgRules {
		_, ok := rules.FindByName(coalesce(cfgRule.Name))
		if ok {
			continue
		}
		status := &RulesStatus{
			RuleName:           coalesce(cfgRule.Name),
			Status:             "NOT DEPLOYED",
			EventPattern:       cfgRule.EventPattern,
			ScheduleExpression: cfgRule.ScheduleExpression,
		}
		rulesStatus = append(rulesStatus, status)
	}
	return rulesStatus, nil
}

func (app *App) newScheduleStatus(ctx context.Context, stateMachineArn string) ([]*ScheduleStatus, error) {
	cfgSchedules := app.cfg.NewSchedules()
	stateMachineQualifiedArn := addQualifierToArn(stateMachineArn, app.StateMachineAliasName())
	schedules, err := app.schedulerSvc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
		StateMachineQualifiedArn: stateMachineQualifiedArn,
		ScheduleNames:            cfgSchedules.Names(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	schedulesStatus := make([]*ScheduleStatus, 0, len(schedules))
	for _, schedule := range schedules {
		status := &ScheduleStatus{
			ScheduleName:               coalesce(schedule.Name),
			ScheduleArn:                coalesce(schedule.ScheduleArn),
			Status:                     string(schedule.State),
			ScheduleExpression:         coalesce(schedule.ScheduleExpression),
			ScheduleExpressionTimezone: coalesce(schedule.ScheduleExpressionTimezone),
		}
		targetQuarifier := strings.TrimPrefix(coalesce(schedule.Target.Arn), stateMachineArn)
		if targetQuarifier == "" {
			targetQuarifier = "$LATEST"
		}
		status.Target = strings.TrimLeft(targetQuarifier, ":")
		schedulesStatus = append(schedulesStatus, status)
	}
	for _, cfgSchedule := range cfgSchedules {
		_, ok := schedules.FindByName(coalesce(cfgSchedule.Name))
		if ok {
			continue
		}
		status := &ScheduleStatus{
			ScheduleName:               coalesce(cfgSchedule.Name),
			Status:                     "NOT DEPLOYED",
			ScheduleExpression:         coalesce(cfgSchedule.ScheduleExpression),
			ScheduleExpressionTimezone: coalesce(cfgSchedule.ScheduleExpressionTimezone),
		}
		schedulesStatus = append(schedulesStatus, status)
	}
	return schedulesStatus, nil
}

type StatusOutput struct {
	StateMachine         *StateMachineStatus `json:"state_machine"`
	EventBridge          []*RulesStatus      `json:"event_bridge,omitempty"`
	EventBridgeScheduler []*ScheduleStatus   `json:"event_bridge_scheduler,omitempty"`
}

func (s *StatusOutput) String() string {
	var builder strings.Builder
	fmt.Fprintln(&builder, "[State Machine]")
	fmt.Fprintln(&builder, s.StateMachine)
	fmt.Fprintln(&builder)
	if len(s.EventBridge) > 0 {
		fmt.Fprintln(&builder, "[EventBridge]")
		for _, rule := range s.EventBridge {
			fmt.Fprintln(&builder, rule)
			fmt.Fprintln(&builder)
		}
	}
	if len(s.EventBridgeScheduler) > 0 {
		fmt.Fprintln(&builder, "[EventBridge Scheduler]")
		for _, schedule := range s.EventBridgeScheduler {
			fmt.Fprintln(&builder, schedule)
			fmt.Fprintln(&builder)
		}
	}
	return builder.String()
}

type StateMachineStatus struct {
	Arn            string `json:"arn"`
	Name           string `json:"name"`
	CurrentVersion int    `json:"current_version"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
}

func (s *StateMachineStatus) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "- Name: %s\n", s.Name)
	fmt.Fprintf(&builder, "  Status: %s\n", s.Status)
	fmt.Fprintf(&builder, "  CurrentVersion: %d\n", s.CurrentVersion)
	fmt.Fprintf(&builder, "  CreatedAt: %s\n", s.CreatedAt)
	fmt.Fprintf(&builder, "  Arn: %s\n", s.Arn)
	return builder.String()
}

type RulesStatus struct {
	RuleArn            string  `json:"rule_arn,omitempty"`
	RuleName           string  `json:"rule_name"`
	Status             string  `json:"status"`
	ScheduleExpression *string `json:"schedule_expression,omitempty"`
	EventPattern       *string `json:"event_pattern,omitempty"`
	Target             string  `json:"target,omitempty"`
}

func (r *RulesStatus) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "- Name: %s\n", r.RuleName)
	fmt.Fprintf(&builder, "  Status: %s\n", r.Status)
	if r.RuleArn != "" {
		fmt.Fprintf(&builder, "  RuleArn: %s\n", r.RuleArn)
	}
	if r.ScheduleExpression != nil {
		fmt.Fprintf(&builder, "  ScheduleExpression: %s\n", *r.ScheduleExpression)
	}
	if r.EventPattern != nil {
		str := *r.EventPattern
		if bs, err := json.Marshal(json.RawMessage([]byte(*r.EventPattern))); err == nil {
			str = string(bs)
		}
		fmt.Fprintf(&builder, "  EventPattern: %s\n", str)
	}
	if r.Target != "" {
		fmt.Fprintf(&builder, "  Target: %s\n", r.Target)
	}
	return builder.String()
}

type ScheduleStatus struct {
	ScheduleName               string `json:"schedule_name"`
	ScheduleArn                string `json:"schedule_arn,omitempty"`
	Status                     string `json:"status"`
	ScheduleExpression         string `json:"schedule_expression"`
	ScheduleExpressionTimezone string `json:"schedule_expression_timezone"`
	Target                     string `json:"target,omitempty"`
}

func (s *ScheduleStatus) String() string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "- Name: %s\n", s.ScheduleName)
	fmt.Fprintf(&builder, "  Status: %s\n", s.Status)
	if s.ScheduleArn != "" {
		fmt.Fprintf(&builder, "  ScheduleArn: %s\n", s.ScheduleArn)
	}
	fmt.Fprintf(&builder, "  ScheduleExpression: %s\n", s.ScheduleExpression)
	if s.ScheduleExpressionTimezone != "" {
		fmt.Fprintf(&builder, "  ScheduleExpressionTimezone: %s\n", s.ScheduleExpressionTimezone)
	}
	if s.Target != "" {
		fmt.Fprintf(&builder, "  Target: %s\n", s.Target)
	}
	return builder.String()
}
