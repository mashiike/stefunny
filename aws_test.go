package stefunny_test

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/mashiike/stefunny"
)

type mockSFnClient struct {
	stefunny.SFnClient
	CreateStateMachineCallCount int
	CreateStateMachineFunc      func(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
}

func (m *mockSFnClient) ResetCallCount() {
	m.CreateStateMachineCallCount = 0
}

func (m *mockSFnClient) CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
	m.CreateStateMachineCallCount++
	if m.CreateStateMachineFunc == nil {
		return nil, errors.New("unexpected Call CreateStateMachine")
	}
	return m.CreateStateMachineFunc(ctx, params, optFns...)
}

type mockCWLogsClient struct {
	stefunny.CWLogsClient
	DescribeLogGroupsCallCount int
	DescribeLogGroupsFunc      func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
}

func (m *mockCWLogsClient) ResetCallCount() {
	m.DescribeLogGroupsCallCount = 0
}

func (m *mockCWLogsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	m.DescribeLogGroupsCallCount++
	if m.DescribeLogGroupsFunc == nil {
		return nil, errors.New("unexpected Call DescribeLogGroups")
	}
	return m.DescribeLogGroupsFunc(ctx, params, optFns...)
}
