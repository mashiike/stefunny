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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
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
				m.sfn.EXPECT().SetAliasName("test").Return()
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
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Hello",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					RuleNames:                []string{},
				}).Return(
					stefunny.EventBridgeRules{},
					nil,
				).Times(1)
				m.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					ScheduleNames:            []string{},
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
				m.sfn.EXPECT().SetAliasName("test").Return()
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
				m.sfn.EXPECT().DeployStateMachine(gomock.Any(), gomock.Cond(
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
				).Times(1)
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Hello",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().DeployRules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Times(1)
				m.scheduler.EXPECT().DeploySchedules(gomock.Any(), "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test", stefunny.Schedules{}, true).Return(
					nil,
				).Times(1)
			},
		},
		{
			casename: "not_found_and_create",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().SetAliasName("test").Return()
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Hello",
				}).Return(
					nil,
					stefunny.ErrStateMachineDoesNotExist,
				).Times(1)
				m.sfn.EXPECT().DeployStateMachine(gomock.Any(), gomock.Cond(
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
				).Times(1)
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Hello",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().DeployRules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Times(1)
				m.scheduler.EXPECT().DeploySchedules(gomock.Any(), "arn:aws:states:us-east-1:000000000000:stateMachine:Hello:test", stefunny.Schedules{}, true).Return(
					nil,
				).Times(1)
			},
		},
		{
			casename: "not_found_and_create_with_event",
			path:     "testdata/event.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().SetAliasName("test").Return()
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Scheduled",
				}).Return(
					nil,
					stefunny.ErrStateMachineDoesNotExist,
				).Times(1)
				m.sfn.EXPECT().DeployStateMachine(gomock.Any(), gomock.Cond(
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
				).Times(1)
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Scheduled",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().DeployRules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					gomock.Cond(
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
										Arn:     aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test"),
										Id:      aws.String("stefunny-managed-state-machine"),
										RoleArn: aws.String("arn:aws:iam::012345678901:role/service-role/Eventbridge-Hello-role"),
									},
								},
							}, input)
						},
					), true).Return(
					nil,
				).Times(1)
				m.scheduler.EXPECT().DeploySchedules(gomock.Any(), "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test", stefunny.Schedules{}, true).Return(
					nil,
				).Times(1)
			},
		},
		{
			casename: "deploy with event",
			path:     "testdata/event.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().SetAliasName("test").Return()
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Scheduled",
				}).Return(
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
				).Times(1)
				m.sfn.EXPECT().DeployStateMachine(gomock.Any(), gomock.Cond(
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
				).Times(1)
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Scheduled",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().DeployRules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					gomock.Cond(
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
										Arn:     aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test"),
										Id:      aws.String("stefunny-managed-state-machine"),
										RoleArn: aws.String("arn:aws:iam::012345678901:role/service-role/Eventbridge-Hello-role"),
									},
								},
							}, input)
						},
					), true).Return(
					nil,
				).Times(1)
				m.scheduler.EXPECT().DeploySchedules(gomock.Any(), "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test", stefunny.Schedules{}, true).Return(
					nil,
				).Times(1)
			},
		},
		{
			casename: "with scheduler",
			path:     "testdata/schedule.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.EXPECT().SetAliasName("test").Return()
				m.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
					Name: "Scheduled",
				}).Return(
					&stefunny.StateMachine{
						CreateStateMachineInput: sfn.CreateStateMachineInput{
							Name:       aws.String("Scheduled"),
							RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
							Definition: aws.String(`{}`),
							Type:       sfntypes.StateMachineTypeStandard,
						},
					},
					nil,
				).Times(1)
				m.sfn.EXPECT().DeployStateMachine(gomock.Any(), gomock.Cond(
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
				).Times(1)
				m.sfn.EXPECT().GetStateMachineArn(gomock.Any(), &stefunny.GetStateMachineArnInput{
					Name: "Scheduled",
				}).Return(
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled",
					nil,
				).Times(2)
				m.eventBridge.EXPECT().DeployRules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					stefunny.EventBridgeRules{},
					true,
				).Return(
					nil,
				).Times(1)
				m.scheduler.EXPECT().DeploySchedules(
					gomock.Any(),
					"arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:test",
					gomock.Cond(
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
					true).Return(
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
			app.SetAliasName("test")
			err := app.Deploy(context.Background(), stefunny.DeployOption{
				DryRun: c.DryRun,
			})
			require.NoError(t, err)
		})
	}
}
