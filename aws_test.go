package stefunny_test

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/mashiike/stefunny"
)

type mockAWSClient struct {
	stefunny.SFnClient
	CallCount                mockClientCallCount
	CreateStateMachineFunc   func(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachineFunc func(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	DeleteStateMachineFunc   func(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	ListStateMachinesFunc    func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error)

	stefunny.CWLogsClient
	DescribeLogGroupsFunc func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
}

type mockClientCallCount struct {
	CreateStateMachine   int
	DescribeStateMachine int
	DeleteStateMachine   int
	DescribeLogGroups    int
	ListStateMachines    int
}

func (m *mockClientCallCount) Reset() {
	m.CreateStateMachine = 0
	m.DescribeStateMachine = 0
	m.DeleteStateMachine = 0
	m.DescribeLogGroups = 0
	m.ListStateMachines = 0
}

func (m *mockAWSClient) CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
	m.CallCount.CreateStateMachine++
	if m.CreateStateMachineFunc == nil {
		return nil, errors.New("unexpected Call CreateStateMachine")
	}
	return m.CreateStateMachineFunc(ctx, params, optFns...)
}

func (m *mockAWSClient) DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	m.CallCount.DescribeStateMachine++
	if m.DescribeStateMachineFunc == nil {
		return nil, errors.New("unexpected Call DescribeStateMachine")
	}
	return m.DescribeStateMachineFunc(ctx, params, optFns...)
}

func (m *mockAWSClient) DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error) {
	m.CallCount.DeleteStateMachine++
	if m.DeleteStateMachineFunc == nil {
		return nil, errors.New("unexpected Call DeleteStateMachine")
	}
	return m.DeleteStateMachineFunc(ctx, params, optFns...)
}
func (m *mockAWSClient) ListStateMachines(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
	m.CallCount.ListStateMachines++
	if m.ListStateMachinesFunc == nil {
		return nil, errors.New("unexpected Call ListStateMachines")
	}
	return m.ListStateMachinesFunc(ctx, params, optFns...)
}

func (m *mockAWSClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	m.CallCount.DescribeLogGroups++
	if m.DescribeLogGroupsFunc == nil {
		return nil, errors.New("unexpected Call DescribeLogGroups")
	}
	return m.DescribeLogGroupsFunc(ctx, params, optFns...)
}
