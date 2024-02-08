package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type SchedulerClient interface {
	CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
}

type SchedulerService interface {
	SearchRelatedSchedules(ctx context.Context, stateMachineArn string) (Schedules, error)
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

func (svc *SchedulerServiceImpl) SearchRelatedSchedules(ctx context.Context, stateMachineArn string) (Schedules, error) {
	log.Printf("[debug] call SearchRelatedSchedules(%s)", stateMachineArn)
	scheduleNames, err := svc.searchRelatedScheduleNames(ctx, stateMachineArn)
	if err != nil {
		return nil, fmt.Errorf("failed to search related schedule names: %w", err)
	}
	schedules := make(Schedules, 0, len(scheduleNames))
	for _, name := range scheduleNames {
		schedule, err := svc.getSchedule(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("failed to get schedule `%s`: %w", name, err)
		}
		schedules = append(schedules, schedule)
	}
	return schedules, nil
}

func (svc *SchedulerServiceImpl) searchRelatedScheduleNames(ctx context.Context, stateMachineArn string) ([]string, error) {
	log.Printf("[debug] call searchRelatedScheduleNames(%s)", stateMachineArn)
	unqualified := unqualifyARN(stateMachineArn)
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
		Arn:          schedule.Arn,
		CreationDate: schedule.CreationDate,
	}
	return result, nil
}

func (svc *SchedulerServiceImpl) DeploySchedules(ctx context.Context, stateMachineArn string, rules Schedules, keepState bool) error {
	return errors.New("not implemented yet")
}
