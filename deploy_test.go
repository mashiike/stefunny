package stefunny_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
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
				m.sfn.On("SetAliasName", "test").Return()
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
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.eventBridge.On("SearchRelatedRules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test").Return(
					stefunny.EventBridgeRules{},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.scheduler.On("SearchRelatedSchedules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test").Return(
					stefunny.Schedules{},
					nil,
				).Once()
			},
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("SetAliasName", "test").Return()
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
				m.sfn.On("DeployStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.Name, "Hello")
					},
				)).Return(
					&stefunny.DeployStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
						UpdateDate:      aws.Time(time.Now()),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.eventBridge.On("DeployRules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.scheduler.On("DeploySchedules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test", stefunny.Schedules{}, false).Return(
					nil,
				).Once()
			},
		},
		{
			casename: "not_found_and_create",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("SetAliasName", "test").Return()
				m.sfn.On("DescribeStateMachine", mock.Anything, "Hello").Return(
					nil,
					stefunny.ErrStateMachineDoesNotExist,
				).Once()
				m.sfn.On("DeployStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.Name, "Hello") &&
							assert.Nil(t, input.StateMachineArn)
					},
				)).Return(
					&stefunny.DeployStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
						UpdateDate:      aws.Time(time.Now()),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.eventBridge.On("DeployRules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Hello").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Once()
				m.scheduler.On("DeploySchedules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test", stefunny.Schedules{}, false).Return(
					nil,
				).Once()
			},
		},
		{
			casename: "not_found_and_create_with_event",
			path:     "testdata/event.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("SetAliasName", "test").Return()
				m.sfn.On("DescribeStateMachine", mock.Anything, "Scheduled").Return(
					nil,
					stefunny.ErrStateMachineDoesNotExist,
				).Once()
				m.sfn.On("DeployStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.Name, "Scheduled") &&
							assert.Nil(t, input.StateMachineArn) &&
							assert.JSONEq(t, *input.Definition, LoadString(t, "testdata/hello_world.asl.json"))
					},
				)).Return(
					&stefunny.DeployStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						UpdateDate:      aws.Time(time.Now()),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.eventBridge.On("DeployRules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					mock.MatchedBy(
						func(input stefunny.EventBridgeRules) bool {
							for _, rule := range input {
								rule.ConfigFilePath = nil
							}
							return assert.EqualValues(t, stefunny.EventBridgeRules{
								{
									PutRuleInput: eventbridge.PutRuleInput{
										Name:               aws.String("Scheduled-hourly"),
										ScheduleExpression: aws.String("rate(1 hour)"),
										RoleArn:            aws.String("arn:aws:iam::012345678901:role/service-role/Eventbridge-Hello-role"),
										Tags: []eventbridgetypes.Tag{
											{
												Key:   aws.String("ManagedBy"),
												Value: aws.String("stefunny"),
											},
										},
									},
									Target: eventbridgetypes.Target{
										Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test"),
										Id:  aws.String("stefunny-managed-state-machine"),
									},
								},
							}, input)
						},
					), true).Return(
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.scheduler.On("DeploySchedules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test", stefunny.Schedules{}, false).Return(
					nil,
				).Once()
			},
		},
		{
			casename: "deploy with event",
			path:     "testdata/event.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("SetAliasName", "test").Return()
				m.sfn.On("DescribeStateMachine", mock.Anything, "Scheduled").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Scheduled"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(LoadString(t, "testdata/hello_world.asl.json")),
						},
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						Status:          sfntypes.StateMachineStatusActive,
						CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
					},
					nil,
				).Once()
				m.sfn.On("DeployStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.Name, "Scheduled") &&
							assert.JSONEq(t, *input.Definition, LoadString(t, "testdata/hello_world.asl.json"))
					},
				)).Return(
					&stefunny.DeployStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						UpdateDate:      aws.Time(time.Now()),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.eventBridge.On("DeployRules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					mock.MatchedBy(
						func(input stefunny.EventBridgeRules) bool {
							for _, rule := range input {
								rule.ConfigFilePath = nil
							}
							return assert.EqualValues(t, stefunny.EventBridgeRules{
								{
									PutRuleInput: eventbridge.PutRuleInput{
										Name:               aws.String("Scheduled-hourly"),
										ScheduleExpression: aws.String("rate(1 hour)"),
										RoleArn:            aws.String("arn:aws:iam::012345678901:role/service-role/Eventbridge-Hello-role"),
										Tags: []eventbridgetypes.Tag{
											{
												Key:   aws.String("ManagedBy"),
												Value: aws.String("stefunny"),
											},
										},
									},
									Target: eventbridgetypes.Target{
										Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test"),
										Id:  aws.String("stefunny-managed-state-machine"),
									},
								},
							}, input)
						},
					), true).Return(
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.scheduler.On("DeploySchedules", mock.Anything, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test", stefunny.Schedules{}, false).Return(
					nil,
				).Once()
			},
		},
		{
			casename: "with scheduler",
			path:     "testdata/schedule.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("SetAliasName", "test").Return()
				m.sfn.On("DescribeStateMachine", mock.Anything, "Scheduled").Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Scheduled"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(`{}`),
							Type:       sfntypes.StateMachineTypeStandard,
						},
					},
					nil,
				).Once()
				m.sfn.On("DeployStateMachine", mock.Anything, mock.MatchedBy(
					func(input *stefunny.StateMachine) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&stefunny.DeployStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
						UpdateDate:      aws.Time(time.Now()),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.eventBridge.On("DeployRules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Once()
				m.sfn.On("GetStateMachineArn", mock.Anything, "Scheduled").Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Once()
				m.scheduler.On(
					"DeploySchedules",
					mock.Anything,
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					mock.MatchedBy(
						func(input stefunny.Schedules) bool {
							for _, schedule := range input {
								schedule.ConfigFilePath = nil
							}
							return assert.EqualValues(t, stefunny.Schedules{
								{
									CreateScheduleInput: scheduler.CreateScheduleInput{
										Name:                       aws.String("Scheduled-hourly"),
										ScheduleExpression:         aws.String("rate(1 hour)"),
										ScheduleExpressionTimezone: aws.String("Asia/Tokyo"),
										Target: &schedulertypes.Target{
											Arn:     aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test"),
											RoleArn: aws.String("arn:aws:iam::012345678901:role/service-role/Eventbridge-Hello-role"),
										},
									},
								},
							}, input)
						},
					),
					false).Return(
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
			err := app.Deploy(context.Background(), stefunny.DeployOption{
				DryRun:    c.DryRun,
				AliasName: "test",
			})
			require.NoError(t, err)
		})
	}
}
