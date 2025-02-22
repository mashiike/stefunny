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
	"github.com/mashiike/stefunny/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSchedulerService__SearchRelatedSchedules(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSchedulerClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListScheduleGroups(gomock.Any(), &scheduler.ListScheduleGroupsInput{
		MaxResults: aws.Int32(100),
	}, gomock.Any()).Return(
		&scheduler.ListScheduleGroupsOutput{
			ScheduleGroups: []schedulertypes.ScheduleGroupSummary{
				{
					Name:  aws.String("default"),
					State: schedulertypes.ScheduleGroupStateActive,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListSchedules(gomock.Any(), &scheduler.ListSchedulesInput{
		MaxResults: aws.Int32(100),
		GroupName:  aws.String("default"),
	}, gomock.Any()).Return(
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
	).Times(1)
	m.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
		Name:      aws.String("Scheduled"),
		GroupName: aws.String("default"),
	}, gomock.Any()).Return(
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
	).Times(1)
	m.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
		Name:      aws.String("Unqualified"),
		GroupName: aws.String("default"),
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
	).Times(1)
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
	ctrl := gomock.NewController(t)
	m := mock.NewMockSchedulerClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListScheduleGroups(gomock.Any(), &scheduler.ListScheduleGroupsInput{
		MaxResults: aws.Int32(100),
	}, gomock.Any()).Return(
		&scheduler.ListScheduleGroupsOutput{
			ScheduleGroups: []schedulertypes.ScheduleGroupSummary{
				{
					Name:  aws.String("default"),
					State: schedulertypes.ScheduleGroupStateActive,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListSchedules(gomock.Any(), &scheduler.ListSchedulesInput{
		MaxResults: aws.Int32(100),
		GroupName:  aws.String("default"),
	}, gomock.Any()).Return(
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
	).Times(1)
	m.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
		Name:      aws.String("Scheduled"),
		GroupName: aws.String("default"),
	}, gomock.Any()).Return(
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
	).Times(1)
	m.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
		Name:      aws.String("Unqualified"),
		GroupName: aws.String("default"),
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
	).Times(1)
	m.EXPECT().GetSchedule(gomock.Any(), &scheduler.GetScheduleInput{
		Name:      aws.String("Monthly"),
		GroupName: aws.String("default"),
	}).Return(
		nil,
		&smithy.GenericAPIError{
			Code: "ResourceNotFoundException",
		},
	).Times(1)
	m.EXPECT().DeleteSchedule(gomock.Any(), &scheduler.DeleteScheduleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&scheduler.DeleteScheduleOutput{},
		nil,
	).Times(1)
	m.EXPECT().UpdateSchedule(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, input *scheduler.UpdateScheduleInput, opts ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error) {
			assert.EqualValues(t, &scheduler.UpdateScheduleInput{
				Name:               aws.String("Scheduled"),
				ScheduleExpression: aws.String("rate(1 hour)"),
				State:              schedulertypes.ScheduleStateDisabled,
				Target: &schedulertypes.Target{
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			}, input)
			return &scheduler.UpdateScheduleOutput{}, nil
		}).Times(1)
	m.EXPECT().CreateSchedule(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, input *scheduler.CreateScheduleInput, opts ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
			assert.EqualValues(t, &scheduler.CreateScheduleInput{
				Name:               aws.String("Monthly"),
				ScheduleExpression: aws.String("cron(0 0 1 * ? *)"),
				State:              schedulertypes.ScheduleStateEnabled,
				Target: &schedulertypes.Target{
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			}, input)
			return &scheduler.CreateScheduleOutput{
				ScheduleArn: aws.String("arn:aws:scheduler:us-east-1:000000000000:schedule:Monthly"),
			}, nil
		}).Times(1)
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
