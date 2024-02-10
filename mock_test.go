package stefunny_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
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
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.DeleteStateMachineOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.DeleteStateMachineOutput")
		return nil, errors.New("mock data is not *sfn.DeleteStateMachineOutput")
	}
	return nil, err
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
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.TagResourceOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.TagResourceOutput")
		return nil, errors.New("mock data is not *sfn.TagResourceOutput")
	}
	return nil, err
}

func (m *mockSFnClient) DescribeExecution(ctx context.Context, params *sfn.DescribeExecutionInput, optFns ...func(*sfn.Options)) (*sfn.DescribeExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.DescribeExecutionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.DescribeExecutionOutput")
		return nil, errors.New("mock data is not *sfn.DescribeExecutionOutput")
	}
	return nil, err
}

func (m *mockSFnClient) StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.StartExecutionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.StartExecutionOutput")
		return nil, errors.New("mock data is not *sfn.StartExecutionOutput")
	}
	return nil, err
}

func (m *mockSFnClient) StartSyncExecution(ctx context.Context, params *sfn.StartSyncExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartSyncExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.StartSyncExecutionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.StartSyncExecutionOutput")
		return nil, errors.New("mock data is not *sfn.StartSyncExecutionOutput")
	}
	return nil, err
}

func (m *mockSFnClient) StopExecution(ctx context.Context, params *sfn.StopExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StopExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.StopExecutionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.StopExecutionOutput")
		return nil, errors.New("mock data is not *sfn.StopExecutionOutput")
	}
	return nil, err
}

func (m *mockSFnClient) GetExecutionHistory(ctx context.Context, params *sfn.GetExecutionHistoryInput, optFns ...func(*sfn.Options)) (*sfn.GetExecutionHistoryOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.GetExecutionHistoryOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.GetExecutionHistoryOutput")
		return nil, errors.New("mock data is not *sfn.GetExecutionHistoryOutput")
	}
	return nil, err
}

func (m *mockSFnClient) CreateStateMachineAlias(ctx context.Context, params *sfn.CreateStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineAliasOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.CreateStateMachineAliasOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.CreateStateMachineAliasOutput")
		return nil, errors.New("mock data is not *sfn.CreateStateMachineAliasOutput")
	}
	return nil, err
}

func (m *mockSFnClient) DescribeStateMachineAlias(ctx context.Context, params *sfn.DescribeStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineAliasOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.DescribeStateMachineAliasOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.DescribeStateMachineAliasOutput")
		return nil, errors.New("mock data is not *sfn.DescribeStateMachineAliasOutput")
	}
	return nil, err
}

func (m *mockSFnClient) ListStateMachineAliases(ctx context.Context, params *sfn.ListStateMachineAliasesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineAliasesOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.ListStateMachineAliasesOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.ListStateMachineAliasesOutput")
		return nil, errors.New("mock data is not *sfn.ListStateMachineAliasesOutput")
	}
	return nil, err
}

func (m *mockSFnClient) ListStateMachineVersions(ctx context.Context, params *sfn.ListStateMachineVersionsInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineVersionsOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.ListStateMachineVersionsOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.ListStateMachineVersionsOutput")
		return nil, errors.New("mock data is not *sfn.ListStateMachineVersionsOutput")
	}
	return nil, err
}

func (m *mockSFnClient) UpdateStateMachineAlias(ctx context.Context, params *sfn.UpdateStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineAliasOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.UpdateStateMachineAliasOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.UpdateStateMachineAliasOutput")
		return nil, errors.New("mock data is not *sfn.UpdateStateMachineAliasOutput")
	}
	return nil, err
}

func (m *mockSFnClient) DeleteStateMachineVersion(ctx context.Context, params *sfn.DeleteStateMachineVersionInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineVersionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*sfn.DeleteStateMachineVersionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *sfn.DeleteStateMachineVersionOutput")
		return nil, errors.New("mock data is not *sfn.DeleteStateMachineVersionOutput")
	}
	return nil, err
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

type mockSchdulerClient struct {
	mock.Mock
	t *testing.T
}

func (m *mockSchdulerClient) CreateSchedule(ctx context.Context, params *scheduler.CreateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.CreateScheduleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*scheduler.CreateScheduleOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *scheduler.CreateScheduleOutput")
		return nil, errors.New("mock data is not *scheduler.CreateScheduleOutput")
	}
	return nil, err
}

func (m *mockSchdulerClient) DeleteSchedule(ctx context.Context, params *scheduler.DeleteScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.DeleteScheduleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*scheduler.DeleteScheduleOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *scheduler.DeleteScheduleOutput")
		return nil, errors.New("mock data is not *scheduler.DeleteScheduleOutput")
	}
	return nil, err
}

func (m *mockSchdulerClient) GetSchedule(ctx context.Context, params *scheduler.GetScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.GetScheduleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*scheduler.GetScheduleOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *scheduler.GetScheduleOutput")
		return nil, errors.New("mock data is not *scheduler.GetScheduleOutput")
	}
	return nil, err
}

func (m *mockSchdulerClient) ListSchedules(ctx context.Context, params *scheduler.ListSchedulesInput, optFns ...func(*scheduler.Options)) (*scheduler.ListSchedulesOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*scheduler.ListSchedulesOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *scheduler.ListSchedulesOutput")
		return nil, errors.New("mock data is not *scheduler.ListSchedulesOutput")
	}
	return nil, err
}

func (m *mockSchdulerClient) UpdateSchedule(ctx context.Context, params *scheduler.UpdateScheduleInput, optFns ...func(*scheduler.Options)) (*scheduler.UpdateScheduleOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*scheduler.UpdateScheduleOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *scheduler.UpdateScheduleOutput")
		return nil, errors.New("mock data is not *scheduler.UpdateScheduleOutput")
	}
	return nil, err
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

func NewMockSchedulerClient(t *testing.T) *mockSchdulerClient {
	t.Helper()
	m := new(mockSchdulerClient)
	m.Test(t)
	return m
}

type mockSFnService struct {
	mock.Mock
	t *testing.T
}

func NewMockSFnService(t *testing.T) *mockSFnService {
	t.Helper()
	m := &mockSFnService{
		t: t,
	}
	m.Test(t)
	return m
}

func (m *mockSFnService) SetAliasName(name string) {
	m.Called(name)
}

func (m *mockSFnService) GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, name, optFns)
	} else {
		args = m.Called(ctx, name)
	}
	return args.String(0), args.Error(1)
}

func (m *mockSFnService) DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*stefunny.StateMachine, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, name, optFns)
	} else {
		args = m.Called(ctx, name)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*stefunny.StateMachine); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *stefunny.StateMachine")
		return nil, errors.New("mock data is not *stefunny.StateMachine")
	}
	return nil, err
}

func (m *mockSFnService) PurgeStateMachineVersions(ctx context.Context, stateMachine *stefunny.StateMachine, keepVersions int, optFns ...func(*sfn.Options)) error {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, keepVersions, optFns)
	} else {
		args = m.Called(ctx, stateMachine, keepVersions)
	}
	return args.Error(0)
}

func (m *mockSFnService) ListStateMachineVersions(ctx context.Context, stateMachine *stefunny.StateMachine, optFns ...func(*sfn.Options)) (*stefunny.ListStateMachineVersionsOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, optFns)
	} else {
		args = m.Called(ctx, stateMachine)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*stefunny.ListStateMachineVersionsOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not []stefunny.StateMachineVersion")
		return nil, errors.New("mock data is not []stefunny.StateMachineVersion")
	}
	return nil, err
}

func (m *mockSFnService) DeployStateMachine(ctx context.Context, stateMachine *stefunny.StateMachine, optFns ...func(*sfn.Options)) (*stefunny.DeployStateMachineOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, optFns)
	} else {
		args = m.Called(ctx, stateMachine)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*stefunny.DeployStateMachineOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *stefunny.DeployStateMachineOutput")
		return nil, errors.New("mock data is not *stefunny.DeployStateMachineOutput")
	}
	return nil, err
}

func (m *mockSFnService) DeleteStateMachine(ctx context.Context, stateMachine *stefunny.StateMachine, optFns ...func(*sfn.Options)) error {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, optFns)
	} else {
		args = m.Called(ctx, stateMachine)
	}
	return args.Error(0)
}

func (m *mockSFnService) RollbackStateMachine(ctx context.Context, stateMachine *stefunny.StateMachine, keepVersion bool, dryRun bool, optFns ...func(*sfn.Options)) error {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, keepVersion, dryRun, optFns)
	} else {
		args = m.Called(ctx, stateMachine, keepVersion, dryRun)
	}
	return args.Error(0)
}

func (m *mockSFnService) StartExecution(ctx context.Context, stateMachine *stefunny.StateMachine, params *stefunny.StartExecutionInput, optFns ...func(*sfn.Options)) (*stefunny.StartExecutionOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, stateMachine, params, optFns)
	} else {
		args = m.Called(ctx, stateMachine, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*stefunny.StartExecutionOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *stefunny.StartExecutionOutput")
		return nil, errors.New("mock data is not *stefunny.StartExecutionOutput")
	}
	return nil, err
}

func (m *mockSFnService) GetExecutionHistory(ctx context.Context, executionArn string) ([]stefunny.HistoryEvent, error) {
	args := m.Called(ctx, executionArn)
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.([]stefunny.HistoryEvent); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not []stefunny.HistoryEvent")
		return nil, errors.New("mock data is not []stefunny.HistoryEvent")
	}
	return nil, err
}

type mockEventBridgeService struct {
	mock.Mock
	t *testing.T
}

func NewMockEventBridgeService(t *testing.T) *mockEventBridgeService {
	t.Helper()
	m := &mockEventBridgeService{
		t: t,
	}
	m.Test(t)
	return m
}

func (m *mockEventBridgeService) SearchRelatedRules(ctx context.Context, params *stefunny.SearchRelatedRulesInput) (stefunny.EventBridgeRules, error) {
	args := m.Called(ctx, params)
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(stefunny.EventBridgeRules); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not stefunny.EventBridgeRules")
		return nil, errors.New("mock data is not stefunny.EventBridgeRules")
	}
	return nil, err
}

func (m *mockEventBridgeService) DeployRules(ctx context.Context, stateMachineARN string, rules stefunny.EventBridgeRules, keepState bool) error {
	args := m.Called(ctx, stateMachineARN, rules, keepState)
	return args.Error(0)
}

type mockSchedulerService struct {
	mock.Mock
	t *testing.T
}

func NewMockSchedulerService(t *testing.T) *mockSchedulerService {
	t.Helper()
	m := &mockSchedulerService{
		t: t,
	}
	m.Test(t)
	return m
}

func (m *mockSchedulerService) SearchRelatedSchedules(ctx context.Context, stateMachineArn string) (stefunny.Schedules, error) {
	args := m.Called(ctx, stateMachineArn)
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(stefunny.Schedules); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not stefunny.Schedules")
		return nil, errors.New("mock data is not stefunny.Schedules")
	}
	return nil, err
}

func (m *mockSchedulerService) DeploySchedules(ctx context.Context, stateMachineARN string, schedules stefunny.Schedules, keepState bool) error {
	args := m.Called(ctx, stateMachineARN, schedules, keepState)
	return args.Error(0)
}

type mocks struct {
	sfn         *mockSFnService
	eventBridge *mockEventBridgeService
	scheduler   *mockSchedulerService
}

func NewMocks(t *testing.T) *mocks {
	t.Helper()
	m := &mocks{
		sfn:         NewMockSFnService(t),
		eventBridge: NewMockEventBridgeService(t),
		scheduler:   NewMockSchedulerService(t),
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
		stefunny.WithSFnService(m.sfn),
		stefunny.WithEventBridgeService(m.eventBridge),
		stefunny.WithSchedulerService(m.scheduler),
	)
	require.NoError(t, err)
	return app
}

type mockCloudWatchLogsClient struct {
	mock.Mock
	t *testing.T
}

func NewMockCloudWatchLogsClient(t *testing.T) *mockCloudWatchLogsClient {
	t.Helper()
	m := &mockCloudWatchLogsClient{
		t: t,
	}
	m.Test(t)
	return m
}

func (m *mockCloudWatchLogsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	var args mock.Arguments
	if len(optFns) > 0 {
		args = m.Called(ctx, params, optFns)
	} else {
		args = m.Called(ctx, params)
	}
	output := args.Get(0)
	err := args.Error(1)
	if err == nil {
		if o, ok := output.(*cloudwatchlogs.DescribeLogGroupsOutput); ok {
			return o, nil
		}
		require.FailNow(m.t, "mock data is not *cloudwatchlogs.DescribeLogGroupsOutput")
		return nil, errors.New("mock data is not *cloudwatchlogs.DescribeLogGroupsOutput")
	}
	return nil, err
}
