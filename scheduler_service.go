package stefunny

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type SchedulerClient interface {
	CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error)
	DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error)
	GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error)
	ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error)
	UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error)
	TagResource(ctx context.Context, params *scheduler.TagResourceInput, optFns ...func(*scheduler.Options)) (*scheduler.TagResourceOutput, error)
	ListTagsForResource(ctx context.Context, params *scheduler.ListTagsForResourceInput, optFns ...func(*scheduler.Options)) (*scheduler.ListTagsForResourceOutput, error)
}

type SchedulerService interface {
	SearchRelatedSchedules(ctx context.Context, stateMachineArn string) (Schedules, error)
	DeploySchedules(ctx context.Context, stateMachineArn string, rules Schedules, keepState bool) error
}

var _ SchedulerService = (*SchedulerServiceImpl)(nil)

type SchedulerServiceImpl struct {
	client              SchedulerClient
	cacheScheduleByName map[string]*scheduler.GetScheduleOutput
	cacheTagsByName     map[string]*scheduler.ListTagsForResourceOutput
}

func NewSchedulerService(client SchedulerClient) *SchedulerServiceImpl {
	return &SchedulerServiceImpl{
		client:              client,
		cacheScheduleByName: make(map[string]*scheduler.GetScheduleOutput),
		cacheTagsByName:     make(map[string]*scheduler.ListTagsForResourceOutput),
	}
}

func (s *SchedulerServiceImpl) SearchRelatedSchedules(ctx context.Context, stateMachineArn string) (Schedules, error) {
	return nil, errors.New("not implemented yet")
}

func (s *SchedulerServiceImpl) DeploySchedules(ctx context.Context, stateMachineArn string, rules Schedules, keepState bool) error {
	return errors.New("not implemented yet")
}
