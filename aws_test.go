package stefunny_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockSFnClient struct {
	mock.Mock
}

type mockEventBridgeClient struct {
	mock.Mock
}

func (m *mockSFnClient) CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.CreateStateMachineOutput), args.Error(1)
}

func (m *mockSFnClient) DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.DescribeStateMachineOutput), args.Error(1)
}

func (m *mockSFnClient) DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.DeleteStateMachineOutput), args.Error(1)
}

func (m *mockSFnClient) ListStateMachines(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.ListStateMachinesOutput), args.Error(1)
}

func (m *mockSFnClient) UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.UpdateStateMachineOutput), args.Error(1)
}

func (m *mockSFnClient) ListTagsForResource(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.ListTagsForResourceOutput), args.Error(1)
}

func (m *mockSFnClient) TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.TagResourceOutput), args.Error(1)
}

func (m *mockSFnClient) DescribeExecution(ctx context.Context, params *sfn.DescribeExecutionInput, optFns ...func(*sfn.Options)) (*sfn.DescribeExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.DescribeExecutionOutput), args.Error(1)
}

func (m *mockSFnClient) StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.StartExecutionOutput), args.Error(1)
}

func (m *mockSFnClient) StartSyncExecution(ctx context.Context, params *sfn.StartSyncExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartSyncExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.StartSyncExecutionOutput), args.Error(1)
}

func (m *mockSFnClient) StopExecution(ctx context.Context, params *sfn.StopExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StopExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.StopExecutionOutput), args.Error(1)
}

func (m *mockSFnClient) GetExecutionHistory(ctx context.Context, params *sfn.GetExecutionHistoryInput, optFns ...func(*sfn.Options)) (*sfn.GetExecutionHistoryOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*sfn.GetExecutionHistoryOutput), args.Error(1)
}

func (m *mockEventBridgeClient) TagResource(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.TagResourceOutput), args.Error(1)
}

func (m *mockEventBridgeClient) PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.PutRuleOutput), args.Error(1)
}
func (m *mockEventBridgeClient) DescribeRule(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.DescribeRuleOutput), args.Error(1)
}

func (m *mockEventBridgeClient) ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.ListTargetsByRuleOutput), args.Error(1)
}

func (m *mockEventBridgeClient) PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.PutTargetsOutput), args.Error(1)
}

func (m *mockEventBridgeClient) DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.DeleteRuleOutput), args.Error(1)
}

func (m *mockEventBridgeClient) RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.RemoveTargetsOutput), args.Error(1)
}

func (m *mockEventBridgeClient) ListTagsForResource(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.ListTagsForResourceOutput), args.Error(1)
}

func (m *mockEventBridgeClient) ListRuleNamesByTarget(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	return args.Get(0).(*eventbridge.ListRuleNamesByTargetOutput), args.Error(1)
}

/*
func getDefaultMock(t *testing.T) *mockAWSClient {
	client := &mockAWSClient{
		CreateStateMachineFunc: func(_ context.Context, params *sfn.CreateStateMachineInput, _ ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
			return &sfn.CreateStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", *params.Name)),
			}, nil
		},
		ListStateMachinesFunc: func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
			return &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{
					newStateMachineListItem("Hello"),
					newStateMachineListItem("Deleting"),
					newStateMachineListItem("Scheduled"),
				},
			}, nil
		},
		DescribeStateMachineFunc: func(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
			parts := strings.Split(*params.StateMachineArn, ":")
			name := parts[len(parts)-1]
			status := sfntypes.StateMachineStatusActive
			if name == "Deleting" {
				status = sfntypes.StateMachineStatusDeleting
			}
			return &sfn.DescribeStateMachineOutput{
				CreationDate:    aws.Time(time.Date(2021, 10, 1, 2, 3, 4, 5, time.UTC)),
				StateMachineArn: params.StateMachineArn,
				Definition:      aws.String(LoadString(t, "testdata/hello_world.asl.json")),
				Status:          status,
				Type:            sfntypes.StateMachineTypeStandard,
				RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::123456789012:role/service-role/StepFunctions-%s-role", name)),
				LoggingConfiguration: &sfntypes.LoggingConfiguration{
					Level: sfntypes.LogLevelOff,
				},
				TracingConfiguration: &sfntypes.TracingConfiguration{
					Enabled: false,
				},
			}, nil
		},
		DeleteStateMachineFunc: func(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error) {
			return &sfn.DeleteStateMachineOutput{}, nil
		},
		UpdateStateMachineFunc: func(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
			return &sfn.UpdateStateMachineOutput{
				UpdateDate: aws.Time(time.Now()),
			}, nil
		},
		SFnTagResourceFunc: func(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error) {
			return &sfn.TagResourceOutput{}, nil
		},
		EBTagResourceFunc: func(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error) {
			return &eventbridge.TagResourceOutput{}, nil
		},
		DescribeRuleFunc: func(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error) {
			if !strings.Contains(*params.Name, "Scheduled") {
				return nil, errors.New("ResourceNotFoundException")
			}
			return &eventbridge.DescribeRuleOutput{
				Name:               aws.String(*params.Name),
				Arn:                aws.String(fmt.Sprintf("arn:aws:events:us-east-1:000000000000:rule/%s", *params.Name)),
				ScheduleExpression: aws.String("rate(1 hour)"),
				CreatedBy:          aws.String("000000000000"),
			}, nil
		},
		DeleteRuleFunc: func(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error) {
			return &eventbridge.DeleteRuleOutput{}, nil
		},
		RemoveTargetsFunc: func(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error) {
			return &eventbridge.RemoveTargetsOutput{}, nil
		},
		ListTargetsByRuleFunc: func(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error) {
			if !strings.Contains(*params.Rule, "Scheduled") {
				return nil, errors.New("ResourceNotFoundException")
			}
			return &eventbridge.ListTargetsByRuleOutput{
				Targets: []eventbridgetypes.Target{
					{
						Id: aws.String("test"),
					},
				},
			}, nil
		},
		SFnListTagsForResourceFunc: func(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error) {
			return &sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}}, nil
		},
		EBListTagsForResourceFunc: func(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error) {
			return &eventbridge.ListTagsForResourceOutput{Tags: []eventbridgetypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
			}}, nil
		},
		ListRuleNamesByTargetFunc: func(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error) {
			if !strings.Contains(*params.TargetArn, "Scheduled") {
				return &eventbridge.ListRuleNamesByTargetOutput{
					RuleNames: []string{},
				}, nil
			}
			return &eventbridge.ListRuleNamesByTargetOutput{
				RuleNames: []string{"test-Scheduled"},
			}, nil
		},
	}
	return client
}
*/

func newListStateMachinesOutput() *sfn.ListStateMachinesOutput {
	return &sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			newStateMachineListItem("Hello"),
			newStateMachineListItem("Scheduled"),
		},
	}
}

func newDescribeStateMachineOutput(name string, deleting bool) *sfn.DescribeStateMachineOutput {
	status := sfntypes.StateMachineStatusActive
	if deleting {
		status = sfntypes.StateMachineStatusDeleting
	}
	return &sfn.DescribeStateMachineOutput{
		CreationDate:    aws.Time(time.Date(2021, 10, 1, 2, 3, 4, 5, time.UTC)),
		StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", name)),
		Definition:      aws.String(""),
		Status:          status,
		Type:            sfntypes.StateMachineTypeStandard,
		RoleArn:         aws.String(fmt.Sprintf("arn:aws:iam::123456789012:role/service-role/StepFunctions-%s-role", name)),
		LoggingConfiguration: &sfntypes.LoggingConfiguration{
			Level: sfntypes.LogLevelOff,
		},
		TracingConfiguration: &sfntypes.TracingConfiguration{
			Enabled: false,
		},
	}
}

func newStateMachineListItem(name string) sfntypes.StateMachineListItem {
	return sfntypes.StateMachineListItem{
		CreationDate:    aws.Time(time.Date(2021, 10, 1, 2, 3, 4, 5, time.UTC)),
		Name:            aws.String(name),
		StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", name)),
	}
}

func newDescribeRuleOutput(name string) *eventbridge.DescribeRuleOutput {
	return &eventbridge.DescribeRuleOutput{}
}

type mocks struct {
	sfn         *mockSFnClient
	eventBridge *mockEventBridgeClient
}

func NewMocks(t *testing.T) *mocks {
	m := &mocks{
		sfn:         new(mockSFnClient),
		eventBridge: new(mockEventBridgeClient),
	}
	m.sfn.Test(t)
	m.eventBridge.Test(t)
	return m
}

func (m *mocks) AssertExpectations(t *testing.T) {
	t.Helper()
	m.sfn.AssertExpectations(t)
	m.eventBridge.AssertExpectations(t)
}

func newMockApp(t *testing.T, path string, m *mocks) *stefunny.App {
	t.Helper()
	l := stefunny.NewConfigLoader(nil, nil)
	ctx := context.Background()
	cfg, err := l.Load(ctx, path)
	require.NoError(t, err)
	app, err := stefunny.New(
		ctx, cfg,
		stefunny.WithSFnClient(m.sfn),
		stefunny.WithEventBridgeClient(m.eventBridge),
	)
	require.NoError(t, err)
	return app
}
