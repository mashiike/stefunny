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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(
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
				).Times(1)
				m.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:current",
				}).Return(
					stefunny.EventBridgeRules{},
					nil,
				).Times(1)
				m.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:current",
				}).Return(
					stefunny.Schedules{},
					nil,
				).Times(1)
			},
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(
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
				).Times(1)
				m.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:current",
				}).Return(
					stefunny.EventBridgeRules{},
					nil,
				).Times(1)
				m.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:current",
				}).Return(
					stefunny.Schedules{},
					nil,
				).Times(1)
				m.sfn.EXPECT().DeleteStateMachine(gomock.Any(), gomock.Cond(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					nil,
				).Times(1)
			},
		},
		{
			casename: "scheduled dry run",
			path:     "testdata/event.yaml",
			DryRun:   true,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Scheduled",
				}).Return(
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
				).Times(1)
				m.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
				}).Return(
					stefunny.EventBridgeRules{
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
							Target: eventbridgetypes.Target{
								Id:      aws.String("stefunny-managed"),
								RoleArn: aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
								Arn:     aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
							},
						},
					},
					nil,
				).Times(1)
				m.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
				}).Return(
					stefunny.Schedules{},
					nil,
				).Times(1)
			},
		},
		{
			casename: "scheduled",
			path:     "testdata/event.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Scheduled",
				}).Return(
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
				).Times(1)
				m.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
				}).Return(
					stefunny.EventBridgeRules{
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
							Target: eventbridgetypes.Target{
								Id:      aws.String("stefunny-managed"),
								RoleArn: aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
								Arn:     aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
							},
						},
					},
					nil,
				).Times(1)
				m.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
				}).Return(
					stefunny.Schedules{},
					nil,
				).Times(1)
				m.sfn.EXPECT().DeleteStateMachine(gomock.Any(), gomock.Cond(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.StateMachineArn, "Scheduled")
					},
				)).Return(
					nil,
				).Times(1)
				m.eventBridge.EXPECT().DeployRules(gomock.Any(), "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current", gomock.Cond(
					func(input stefunny.EventBridgeRules) bool {
						return len(input) == 0
					},
				), false).Return(
					nil,
				).Times(1)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			mocks := NewMocks(t)
			defer mocks.Finish()
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
