package stefunny_test

import (
	"context"
	"errors"
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
	t *testing.T
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
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.CreateStateMachineOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.CreateStateMachineOutput")
		return nil, errors.New("mock data is not *sfn.CreateStateMachineOutput")
	}
	return nil, err
}

func (m *mockSFnClient) DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.DescribeStateMachineOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.DescribeStateMachineOutput")
		return nil, errors.New("mock data is not *sfn.DescribeStateMachineOutput")
	}
	return nil, err
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
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.ListStateMachinesOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.ListStateMachinesOutput")
		return nil, errors.New("mock data is not *sfn.ListStateMachinesOutput")
	}
	return nil, err
}

func (m *mockSFnClient) UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.UpdateStateMachineOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.UpdateStateMachineOutput")
		return nil, errors.New("mock data is not *sfn.UpdateStateMachineOutput")
	}
	return nil, err
}

func (m *mockSFnClient) ListTagsForResource(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.ListTagsForResourceOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.ListTagsForResourceOutput")
		return nil, errors.New("mock data is not *sfn.ListTagsForResourceOutput")
	}
	return nil, err
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

func NewMockSFnClient(t *testing.T) *mockSFnClient {
	t.Helper()
	m := &mockSFnClient{
		t: t,
	}
	m.Test(t)
	return m
}

func NewMockEventBridgeClient(t *testing.T) *mockEventBridgeClient {
	t.Helper()
	m := new(mockEventBridgeClient)
	m.Test(t)
	return m
}

type mocks struct {
	sfn         *mockSFnClient
	eventBridge *mockEventBridgeClient
}

func NewMocks(t *testing.T) *mocks {
	t.Helper()
	m := &mocks{
		sfn:         NewMockSFnClient(t),
		eventBridge: NewMockEventBridgeClient(t),
	}
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
