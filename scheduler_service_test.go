package stefunny_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/smithy-go"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSchedulerService__SearchRelatedSchedules(t *testing.T) {
	LoggerSetup(t, "debug")
	m := NewMockSchedulerClient(t)
	defer m.AssertExpectations(t)

	m.On("ListSchedules", mock.Anything, &scheduler.ListSchedulesInput{
		MaxResults: aws.Int32(100),
	}).Return(
		&scheduler.ListSchedulesOutput{
			Schedules: []schedulertypes.ScheduleSummary{
				{
					Name: aws.String("Scheduled"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
					},
				},
				{
					Name: aws.String("Hoge"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:lambda:us-east-1:000000000000:function:Hoge"),
					},
				},
				{
					Name: aws.String("Unqualified"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
					},
				},
			},
		},
		nil,
	).Once()
	m.On("GetSchedule", mock.Anything, &scheduler.GetScheduleInput{
		Name: aws.String("Scheduled"),
	}).Return(
		&scheduler.GetScheduleOutput{
			Name:               aws.String("Scheduled"),
			ScheduleExpression: aws.String("rate(1 day)"),
			State:              schedulertypes.ScheduleStateEnabled,
			Target: &schedulertypes.Target{
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
			},
			Arn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Scheduled"),
		},
		nil,
	).Once()
	m.On("GetSchedule", mock.Anything, &scheduler.GetScheduleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&scheduler.GetScheduleOutput{
			Name:               aws.String("Unqualified"),
			ScheduleExpression: aws.String("rate(1 day)"),
			State:              schedulertypes.ScheduleStateEnabled,
			Target: &schedulertypes.Target{
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
			},
			Arn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Unqualified"),
		},
		nil,
	).Once()
	svc := stefunny.NewSchedulerService(m)
	ctx := context.Background()
	schedules, err := svc.SearchRelatedSchedules(ctx, &stefunny.SearchRelatedSchedulesInput{
		StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
	})
	require.NoError(t, err)
	require.EqualValues(t,
		stefunny.Schedules{
			{
				CreateScheduleInput: scheduler.CreateScheduleInput{
					Name:               aws.String("Scheduled"),
					ScheduleExpression: aws.String("rate(1 day)"),
					State:              schedulertypes.ScheduleStateEnabled,
					Target: &schedulertypes.Target{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
					},
				},
				ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Scheduled"),
			},
			{
				CreateScheduleInput: scheduler.CreateScheduleInput{
					Name:               aws.String("Unqualified"),
					ScheduleExpression: aws.String("rate(1 day)"),
					State:              schedulertypes.ScheduleStateEnabled,
					Target: &schedulertypes.Target{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
					},
				},
				ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Unqualified"),
			},
		},
		schedules,
	)
}

func TestSchedulerService__DeploySchedules(t *testing.T) {
	LoggerSetup(t, "debug")
	m := NewMockSchedulerClient(t)
	defer m.AssertExpectations(t)

	m.On("ListSchedules", mock.Anything, &scheduler.ListSchedulesInput{
		MaxResults: aws.Int32(100),
	}).Return(
		&scheduler.ListSchedulesOutput{
			Schedules: []schedulertypes.ScheduleSummary{
				{
					Name: aws.String("Scheduled"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
					},
				},
				{
					Name: aws.String("Hoge"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:lambda:us-east-1:000000000000:function:Hoge"),
					},
				},
				{
					Name: aws.String("Unqualified"),
					Target: &schedulertypes.TargetSummary{
						Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
					},
				},
			},
		},
		nil,
	).Once()
	m.On("GetSchedule", mock.Anything, &scheduler.GetScheduleInput{
		Name: aws.String("Scheduled"),
	}).Return(
		&scheduler.GetScheduleOutput{
			Name:               aws.String("Scheduled"),
			ScheduleExpression: aws.String("rate(1 day)"),
			State:              schedulertypes.ScheduleStateDisabled,
			Target: &schedulertypes.Target{
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
			},
			Arn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Scheduled"),
		},
		nil,
	).Once()
	m.On("GetSchedule", mock.Anything, &scheduler.GetScheduleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&scheduler.GetScheduleOutput{
			Name:               aws.String("Unqualified"),
			ScheduleExpression: aws.String("rate(1 day)"),
			State:              schedulertypes.ScheduleStateEnabled,
			Target: &schedulertypes.Target{
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
			},
			Arn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Unqualified"),
		},
		nil,
	).Once()
	m.On("GetSchedule", mock.Anything, &scheduler.GetScheduleInput{
		Name: aws.String("Monthly"),
	}).Return(
		nil,
		&smithy.GenericAPIError{
			Code: "ResourceNotFoundException",
		},
	).Once()
	m.On("DeleteSchedule", mock.Anything, &scheduler.DeleteScheduleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&scheduler.DeleteScheduleOutput{},
		nil,
	).Once()
	m.On("UpdateSchedule", mock.Anything, mock.MatchedBy(
		func(input *scheduler.UpdateScheduleInput) bool {
			return assert.EqualValues(t, &scheduler.UpdateScheduleInput{
				Name:               aws.String("Scheduled"),
				ScheduleExpression: aws.String("rate(1 hour)"),
				State:              schedulertypes.ScheduleStateDisabled,
				Target: &schedulertypes.Target{
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			}, input)
		})).Return(
		&scheduler.UpdateScheduleOutput{},
		nil,
	).Once()
	m.On("CreateSchedule", mock.Anything, mock.MatchedBy(
		func(input *scheduler.CreateScheduleInput) bool {
			return assert.EqualValues(t, &scheduler.CreateScheduleInput{
				Name:               aws.String("Monthly"),
				ScheduleExpression: aws.String("cron(0 0 1 * ? *)"),
				State:              schedulertypes.ScheduleStateEnabled,
				Target: &schedulertypes.Target{
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			}, input)
		},
	)).Return(
		&scheduler.CreateScheduleOutput{
			ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Monthly"),
		},
		nil,
	).Once()
	svc := stefunny.NewSchedulerService(m)
	ctx := context.Background()
	err := svc.DeploySchedules(ctx, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current", stefunny.Schedules{
		{
			CreateScheduleInput: scheduler.CreateScheduleInput{
				Name:               aws.String("Scheduled"),
				ScheduleExpression: aws.String("rate(1 hour)"),
				State:              schedulertypes.ScheduleStateEnabled,
				Target:             &schedulertypes.Target{},
			},
		},
		{
			CreateScheduleInput: scheduler.CreateScheduleInput{
				Name:               aws.String("Monthly"),
				ScheduleExpression: aws.String("cron(0 0 1 * ? *)"),
				State:              schedulertypes.ScheduleStateEnabled,
				Target:             &schedulertypes.Target{},
			},
		},
		{
			CreateScheduleInput: scheduler.CreateScheduleInput{
				Name:                       aws.String("Past"),
				ScheduleExpression:         aws.String("at(1900-01-01T00:00:00)"),
				ScheduleExpressionTimezone: aws.String("UTC"),
				State:                      schedulertypes.ScheduleStateEnabled,
				Target:                     &schedulertypes.Target{},
			},
		},
		{
			CreateScheduleInput: scheduler.CreateScheduleInput{
				Name:               aws.String("OverEndDate"),
				ScheduleExpression: aws.String("rate(1 day)"),
				State:              schedulertypes.ScheduleStateEnabled,
				Target:             &schedulertypes.Target{},
				EndDate:            aws.Time(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
		},
	}, true)
	require.NoError(t, err)
}
