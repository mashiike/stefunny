package stefunny_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	cases := []struct {
		casename   string
		path       string
		DryRun     bool
		setupMocks func(*testing.T, *mocks)
	}{
		{
			casename: "default_config dryrun",
			path:     "testdata/stefunny.yaml",
			DryRun:   true,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("DescribeStateMachine", mock.Anything, "Hello").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:    aws.String("Hello"),
							RoleArn: aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
						},
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
						Status:          sfntypes.StateMachineStatusActive,
						CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
					nil,
				).Once()
				m.eventBridge.On("SearchScheduleRule", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello").Return(
					stefunny.ScheduleRules{},
					nil,
				).Once()
			},
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("DescribeStateMachine", mock.Anything, "Hello").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Hello"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(`{}`),
						},
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
						Status:          sfntypes.StateMachineStatusActive,
						CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
					nil,
				).Once()
				m.eventBridge.On("SearchScheduleRule", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello").Return(
					stefunny.ScheduleRules{},
					nil,
				).Once()
				m.sfn.On("DeleteStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					nil,
				).Once()
			},
		},
		{
			casename: "scheduled dry run",
			path:     "testdata/schedule.yaml",
			DryRun:   true,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("DescribeStateMachine", mock.Anything, "Scheduled").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Hello"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(`{}`),
						},
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						Status:          sfntypes.StateMachineStatusActive,
						CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
					nil,
				).Once()
				m.eventBridge.On("SearchScheduleRule", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled").Return(
					stefunny.ScheduleRules{
						{
							PutRuleInput: eventbridge.PutRuleInput{
								Name: aws.String("Scheduled"),
								Tags: []eventbridgetypes.Tag{
									{
										Key:   aws.String("ManagedBy"),
										Value: aws.String("stefunny"),
									},
								},
							},
							TargetRoleArn: "arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1",
							Targets: []eventbridgetypes.Target{
								{
									Id:  aws.String("Scheduled"),
									Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
								},
							},
						},
					},
					nil,
				).Once()
			},
		},
		{
			casename: "scheduled",
			path:     "testdata/schedule.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("DescribeStateMachine", mock.Anything, "Scheduled").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Hello"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(`{}`),
						},
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						Status:          sfntypes.StateMachineStatusActive,
						CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
					nil,
				).Once()
				m.eventBridge.On("SearchScheduleRule", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled").Return(
					stefunny.ScheduleRules{
						{
							PutRuleInput: eventbridge.PutRuleInput{
								Name: aws.String("Scheduled"),
								Tags: []eventbridgetypes.Tag{
									{
										Key:   aws.String("ManagedBy"),
										Value: aws.String("stefunny"),
									},
								},
							},
							TargetRoleArn: "arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1",
							Targets: []eventbridgetypes.Target{
								{
									Id:  aws.String("Scheduled"),
									Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
								},
							},
						},
					},
					nil,
				).Once()
				m.sfn.On("DeleteStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.StateMachineArn, "Scheduled")
					},
				)).Return(
					nil,
				).Once()
				m.eventBridge.On("DeleteScheduleRules", mock.Anything, mock.MatchedBy(
					func(input stefunny.ScheduleRules) bool {
						return assert.Contains(t, *input[0].Name, "Scheduled")
					},
				)).Return(
					nil,
				).Once()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			mocks := NewMocks(t)
			defer mocks.AssertExpectations(t)
			if c.setupMocks != nil {
				c.setupMocks(t, mocks)
			}
			app := newMockApp(t, c.path, mocks)
			err := app.Delete(context.Background(), stefunny.DeleteOption{
				DryRun: c.DryRun,
				Force:  true,
			})
			require.NoError(t, err)
		})
	}
}
