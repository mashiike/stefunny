package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/smithy-go"
)

var ErrScheduleNotFound = errors.New("schedule not found")

type SchedulerClient interface {
	CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
	ListScheduleGroups(ctx context.Context, params *scheduler.ListScheduleGroupsInput, optFns ...func(*scheduler.Options)) (*scheduler.ListScheduleGroupsOutput, error)
	UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
}

type SchedulerService interface {
	SearchRelatedSchedules(ctx context.Context, params *SearchRelatedSchedulesInput) (Schedules, error)
	DeploySchedules(ctx context.Context, stateMachineArn string, rules Schedules, keepState bool) error
}

var _ SchedulerService = (*SchedulerServiceImpl)(nil)

type SchedulerServiceImpl struct {
	client                      SchedulerClient
	cacheNamesByStateMachineArn map[string][]string
	cacheScheduleByName         map[string]*scheduler.GetScheduleOutput
	cacheGroups                 []schedulertypes.ScheduleGroupSummary
}

func NewSchedulerService(client SchedulerClient) *SchedulerServiceImpl {
	return &SchedulerServiceImpl{
		client:                      client,
		cacheNamesByStateMachineArn: make(map[string][]string),
		cacheScheduleByName:         make(map[string]*scheduler.GetScheduleOutput),
	}
}

type SearchRelatedSchedulesInput struct {
	StateMachineQualifiedArn string
	ScheduleNames            []string
}

func (svc *SchedulerServiceImpl) SearchRelatedSchedules(ctx context.Context, params *SearchRelatedSchedulesInput) (Schedules, error) {
	log.Printf("[debug] call SearchRelatedSchedules(%#v)", params)
	stateMachineArn := params.StateMachineQualifiedArn
	scheduleNames, err := svc.searchRelatedScheduleNames(ctx, stateMachineArn)
	if err != nil {
		return nil, fmt.Errorf("failed to search related schedule names: %w", err)
	}
	if len(params.ScheduleNames) > 0 {
		scheduleNames = append(scheduleNames, params.ScheduleNames...)
		scheduleNames = unique(scheduleNames)
	}
	schedules := make(Schedules, 0, len(scheduleNames))
	for _, name := range scheduleNames {
		schedule, err := svc.getSchedule(ctx, name)
		if err != nil {
			if !errors.Is(err, ErrScheduleNotFound) {
				return nil, fmt.Errorf("failed to get schedule `%s`: %w", name, err)
			}
			continue
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func (svc *SchedulerServiceImpl) searchRelatedScheduleNames(ctx context.Context, stateMachineArn string) ([]string, error) {
	log.Printf("[debug] call searchRelatedScheduleNames(%s)", stateMachineArn)
	unqualified := removeQualifierFromArn(stateMachineArn)
	log.Printf("[debug] state machine arn is `%s`", stateMachineArn)
	log.Printf("[debug] unqualified state machine arn is `%s`", unqualified)
	names, ok := svc.cacheNamesByStateMachineArn[stateMachineArn]
	if ok {
		if unqualified != stateMachineArn {
			unqualifiedNames, ok := svc.cacheNamesByStateMachineArn[unqualified]
			if !ok {
				log.Println("[warn] unqualified state machine Arn is not found in cache")
			}
			names = append(names, unqualifiedNames...)
			names = unique(names)
		}
		return names, nil
	}
	unqualifiedNames := make([]string, 0, 100)
	err := svc.forEachGroups(ctx, func(ctx context.Context, group schedulertypes.ScheduleGroupSummary) error {
		p := scheduler.NewListSchedulesPaginator(svc.client, &scheduler.ListSchedulesInput{
			GroupName:  group.Name,
			MaxResults: aws.Int32(100),
		})
		for p.HasMorePages() {
			page, err := p.NextPage(ctx)
			if err != nil {
				return fmt.Errorf("failed to list schedules: %w", err)
			}
			for _, schedule := range page.Schedules {
				targetArn := coalesce(schedule.Target.Arn)
				log.Printf("[debug] schedule `%s` target Arn is `%s`", coalesce(schedule.Name), targetArn)
				if targetArn == stateMachineArn {
					names = append(names, coalesce(schedule.Name))
				}
				if targetArn == unqualified {
					unqualifiedNames = append(unqualifiedNames, coalesce(schedule.Name))
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	names = unique(names)
	svc.cacheNamesByStateMachineArn[stateMachineArn] = names
	if unqualified == stateMachineArn {
		return names, nil
	}
	svc.cacheNamesByStateMachineArn[unqualified] = unqualifiedNames
	result := make([]string, 0, len(names)+len(unqualifiedNames))
	result = append(result, names...)
	result = append(result, unqualifiedNames...)
	return unique(result), nil
}

func (svc *SchedulerServiceImpl) forEachGroups(ctx context.Context, fn func(ctx context.Context, group schedulertypes.ScheduleGroupSummary) error) error {
	if len(svc.cacheGroups) > 0 {
		for _, group := range svc.cacheGroups {
			if err := fn(ctx, group); err != nil {
				return err
			}
		}
		return nil
	}
	gp := scheduler.NewListScheduleGroupsPaginator(svc.client, &scheduler.ListScheduleGroupsInput{
		MaxResults: aws.Int32(100),
	})
	groupSummaries := make([]schedulertypes.ScheduleGroupSummary, 0, 100)
	for gp.HasMorePages() {
		groups, err := gp.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list schedule groups: %w", err)
		}
		for _, group := range groups.ScheduleGroups {
			if err := fn(ctx, group); err != nil {
				return err
			}
			groupSummaries = append(groupSummaries, group)
		}
	}
	svc.cacheGroups = groupSummaries
	return nil
}

func (svc *SchedulerServiceImpl) getSchedule(ctx context.Context, name string) (*Schedule, error) {
	log.Printf("[debug] call getSchedule(%s)", name)
	schedule, ok := svc.cacheScheduleByName[name]
	if !ok {
		var found bool
		err := svc.forEachGroups(ctx, func(ctx context.Context, group schedulertypes.ScheduleGroupSummary) error {
			var err error
			if found {
				return nil
			}
			if group.State != schedulertypes.ScheduleGroupStateActive {
				return nil
			}
			schedule, err = svc.client.GetSchedule(ctx, &scheduler.GetScheduleInput{
				Name:      aws.String(name),
				GroupName: group.Name,
			})
			if err != nil {
				var apiErr smithy.APIError
				if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceNotFoundException" {
					return nil
				}
				return fmt.Errorf("scheduler.GetSchedule `%s`: %w", name, err)
			}
			found = true
			return nil
		})
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, ErrScheduleNotFound
		}
		svc.cacheScheduleByName[name] = schedule
	}
	result := &Schedule{
		CreateScheduleInput: scheduler.CreateScheduleInput{
			Name:                       schedule.Name,
			FlexibleTimeWindow:         schedule.FlexibleTimeWindow,
			ScheduleExpression:         schedule.ScheduleExpression,
			ScheduleExpressionTimezone: schedule.ScheduleExpressionTimezone,
			State:                      schedule.State,
			Target:                     schedule.Target,
			ActionAfterCompletion:      schedule.ActionAfterCompletion,
			Description:                schedule.Description,
			EndDate:                    schedule.EndDate,
			GroupName:                  schedule.GroupName,
			StartDate:                  schedule.StartDate,
			KmsKeyArn:                  schedule.KmsKeyArn,
		},
		ScheduleArn:  schedule.Arn,
		CreationDate: schedule.CreationDate,
	}
	return result, nil
}

func (svc *SchedulerServiceImpl) DeploySchedules(ctx context.Context, stateMachineArn string, schedules Schedules, keepState bool) error {
	newSchedules, passed := schedules.FilterPassed()
	for _, schedule := range passed {
		log.Printf("[warn] schedule `%s` has passed, skip deploy or delete", coalesce(schedule.Name))
	}
	currentSchedules, err := svc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
		StateMachineQualifiedArn: stateMachineArn,
		ScheduleNames:            newSchedules.Names(),
	})
	if err != nil {
		return fmt.Errorf("failed to search related schedules: %w", err)
	}
	if keepState {
		schedules.SyncState(currentSchedules)
	}
	newSchedules.SetStateMachineQualifiedArn(stateMachineArn)
	plan := sliceDiff(currentSchedules, newSchedules, func(schedule *Schedule) string {
		return coalesce(schedule.Name)
	})
	for _, schedule := range plan.Delete {
		log.Println("[info] delete schedule", coalesce(schedule.ScheduleArn))
		_, err := svc.client.DeleteSchedule(ctx, &scheduler.DeleteScheduleInput{
			Name: schedule.Name,
		})
		if err != nil {
			return fmt.Errorf("failed to delete schedule `%s`: %w", coalesce(schedule.Name), err)
		}
	}
	for _, schedule := range plan.Change {
		log.Println("[info] update schedule", coalesce(schedule.Before.ScheduleArn))
		if _, err := svc.client.UpdateSchedule(ctx, &scheduler.UpdateScheduleInput{
			Name:                       schedule.After.Name,
			FlexibleTimeWindow:         schedule.After.FlexibleTimeWindow,
			ScheduleExpression:         schedule.After.ScheduleExpression,
			ScheduleExpressionTimezone: schedule.After.ScheduleExpressionTimezone,
			State:                      schedule.After.State,
			Target:                     schedule.After.Target,
			ActionAfterCompletion:      schedule.After.ActionAfterCompletion,
			Description:                schedule.After.Description,
			EndDate:                    schedule.After.EndDate,
			GroupName:                  schedule.After.GroupName,
			StartDate:                  schedule.After.StartDate,
			KmsKeyArn:                  schedule.After.KmsKeyArn,
		}); err != nil {
			return fmt.Errorf("failed to update schedule `%s`: %w", coalesce(schedule.Before.Name), err)
		}
	}
	for _, schedule := range plan.Add {
		log.Println("[info] create schedule", coalesce(schedule.Name))
		output, err := svc.client.CreateSchedule(ctx, &schedule.CreateScheduleInput)
		if err != nil {
			return fmt.Errorf("failed to create schedule `%s`: %w", coalesce(schedule.Name), err)
		}
		schedule.ScheduleArn = output.ScheduleArn
	}
	return nil
}
