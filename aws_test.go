package stefunny_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
)

type mockAWSClient struct {
	stefunny.SFnClient
	CallCount                mockClientCallCount
	CreateStateMachineFunc   func(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachineFunc func(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	DeleteStateMachineFunc   func(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	ListStateMachinesFunc    func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error)
	UpdateStateMachineFunc   func(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)

	stefunny.CWLogsClient
	DescribeLogGroupsFunc func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)
}

type mockClientCallCount struct {
	CreateStateMachine   int
	DescribeStateMachine int
	DeleteStateMachine   int
	DescribeLogGroups    int
	ListStateMachines    int
	UpdateStateMachine   int
}

func (m *mockClientCallCount) Reset() {
	m.CreateStateMachine = 0
	m.DescribeStateMachine = 0
	m.DeleteStateMachine = 0
	m.DescribeLogGroups = 0
	m.ListStateMachines = 0
	m.UpdateStateMachine = 0
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

func (m *mockAWSClient) UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
	m.CallCount.UpdateStateMachine++
	if m.UpdateStateMachineFunc == nil {
		return nil, errors.New("unexpected Call UpdateStateMachine")
	}
	return m.UpdateStateMachineFunc(ctx, params, optFns...)
}

func (m *mockAWSClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	m.CallCount.DescribeLogGroups++
	if m.DescribeLogGroupsFunc == nil {
		return nil, errors.New("unexpected Call DescribeLogGroups")
	}
	return m.DescribeLogGroupsFunc(ctx, params, optFns...)
}

func getDefaultMock(t *testing.T) *mockAWSClient {
	client := &mockAWSClient{
		CreateStateMachineFunc: func(_ context.Context, params *sfn.CreateStateMachineInput, _ ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
			return &sfn.CreateStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", *params.Name)),
			}, nil
		},
		DescribeLogGroupsFunc: func(_ context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, _ ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
			return &cloudwatchlogs.DescribeLogGroupsOutput{
				LogGroups: []logstypes.LogGroup{
					{
						LogGroupName: params.LogGroupNamePrefix,
						Arn:          aws.String("arn:aws:logs:us-east-1:123456789012:log-group:" + *params.LogGroupNamePrefix),
					},
				},
			}, nil
		},
		ListStateMachinesFunc: func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
			return &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{
					newStateMachineListItem("Hello"),
					newStateMachineListItem("Deleting"),
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
				Definition:      aws.String(testutil.LoadString(t, "testdata/hello_world.asl.json")),
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
	}
	return client
}

func newStateMachineListItem(name string) sfntypes.StateMachineListItem {
	return sfntypes.StateMachineListItem{
		CreationDate:    aws.Time(time.Date(2021, 10, 1, 2, 3, 4, 5, time.UTC)),
		Name:            aws.String(name),
		StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", name)),
	}
}
