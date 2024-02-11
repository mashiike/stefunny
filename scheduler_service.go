package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/smithy-go"
)

var ErrScheduleNotFound = errors.New("schedule not found")

type SchedulerClient interface {
	CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
}

type SchedulerService interface {
	SearchRelatedSchedules(ctx context.Context, params *SearchRelatedSchedulesInput) (Schedules, error)
	DeploySchedules(ctx context.Context, stateMachineArn string, rules Schedules, keepState bool) error
}

var _ SchedulerService = (*SchedulerServiceImpl)(nil)

type SchedulerServiceImpl struct {
	client                      SchedulerClient
	cacheNamesByStateMachineARN map[string][]string
	cacheScheduleByName         map[string]*scheduler.GetScheduleOutput
}

func NewSchedulerService(client SchedulerClient) *SchedulerServiceImpl {
	return &SchedulerServiceImpl{
		client:                      client,
		cacheNamesByStateMachineARN: make(map[string][]string),
		cacheScheduleByName:         make(map[string]*scheduler.GetScheduleOutput),
	}
}

type SearchRelatedSchedulesInput struct {
	StateMachineQualifiedARN string
	ScheduleNames            []string
}

func (svc *SchedulerServiceImpl) SearchRelatedSchedules(ctx context.Context, params *SearchRelatedSchedulesInput) (Schedules, error) {

	log.Printf("[debug] call SearchRelatedSchedules(%#v)", params)
	stateMachineArn := params.StateMachineQualifiedARN
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
	names, ok := svc.cacheNamesByStateMachineARN[stateMachineArn]
	if ok {
		if unqualified != stateMachineArn {
			unqualifiedNames, ok := svc.cacheNamesByStateMachineARN[unqualified]
			if !ok {
				log.Println("[warn] unqualified state machine ARN is not found in cache")
			}
			names = append(names, unqualifiedNames...)
			names = unique(names)
		}
		return names, nil
	}
	p := scheduler.NewListSchedulesPaginator(svc.client, &scheduler.ListSchedulesInput{
		MaxResults: aws.Int32(100),
	})
	unqualifiedNames := make([]string, 0, 100)
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list schedules: %w", err)
		}
		for _, schedule := range page.Schedules {
			targetARN := coalesce(schedule.Target.Arn)
			log.Printf("[debug] schedule `%s` target ARN is `%s`", coalesce(schedule.Name), targetARN)
			if targetARN == stateMachineArn {
				names = append(names, coalesce(schedule.Name))
			}
			if targetARN == unqualified {
				unqualifiedNames = append(unqualifiedNames, coalesce(schedule.Name))
			}
		}
	}
	names = unique(names)
	svc.cacheNamesByStateMachineARN[stateMachineArn] = names
	if unqualified == stateMachineArn {
		return names, nil
	}
	svc.cacheNamesByStateMachineARN[unqualified] = unqualifiedNames
	result := make([]string, 0, len(names)+len(unqualifiedNames))
	result = append(result, names...)
	result = append(result, unqualifiedNames...)
	return unique(result), nil
}

func (svc *SchedulerServiceImpl) getSchedule(ctx context.Context, name string) (*Schedule, error) {
	log.Printf("[debug] call getSchedule(%s)", name)
	schedule, ok := svc.cacheScheduleByName[name]
	if !ok {
		var err error
		schedule, err = svc.client.GetSchedule(ctx, &scheduler.GetScheduleInput{
			Name: aws.String(name),
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) && apiErr.ErrorCode() == "ResourceNotFoundException" {
				return nil, ErrScheduleNotFound
			}
			return nil, fmt.Errorf("scheduler.GetSchedule `%s`: %w", name, err)
		}
		svc.cacheScheduleByName[name] = schedule
	}
	result := &Schedule{
		CreateScheduleInput: scheduler.CreateScheduleInput{
			Name:                  schedule.Name,
			FlexibleTimeWindow:    schedule.FlexibleTimeWindow,
			ScheduleExpression:    schedule.ScheduleExpression,
			State:                 schedule.State,
			Target:                schedule.Target,
			ActionAfterCompletion: schedule.ActionAfterCompletion,
			Description:           schedule.Description,
			EndDate:               schedule.EndDate,
			GroupName:             schedule.GroupName,
			StartDate:             schedule.StartDate,
			KmsKeyArn:             schedule.KmsKeyArn,
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
		StateMachineQualifiedARN: stateMachineArn,
		ScheduleNames:            newSchedules.Names(),
	})
	if err != nil {
		return fmt.Errorf("failed to search related schedules: %w", err)
	}
	if keepState {
		schedules.SyncState(currentSchedules)
	}
	newSchedules.SetStateMachineQualifiedARN(stateMachineArn)
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
			Name:                  schedule.After.Name,
			FlexibleTimeWindow:    schedule.After.FlexibleTimeWindow,
			ScheduleExpression:    schedule.After.ScheduleExpression,
			State:                 schedule.After.State,
			Target:                schedule.After.Target,
			ActionAfterCompletion: schedule.After.ActionAfterCompletion,
			Description:           schedule.After.Description,
			EndDate:               schedule.After.EndDate,
			GroupName:             schedule.After.GroupName,
			StartDate:             schedule.After.StartDate,
			KmsKeyArn:             schedule.After.KmsKeyArn,
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
