package stefunny_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

type mockAWSClient struct {
	CallCount                  mockClientCallCount
	CreateStateMachineFunc     func(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachineFunc   func(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	DeleteStateMachineFunc     func(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	ListStateMachinesFunc      func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error)
	UpdateStateMachineFunc     func(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)
	SFnListTagsForResourceFunc func(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error)
	SFnTagResourceFunc         func(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error)

	DescribeLogGroupsFunc func(context.Context, *cloudwatchlogs.DescribeLogGroupsInput, ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error)

	PutRuleFunc               func(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	DescribeRuleFunc          func(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error)
	ListTargetsByRuleFunc     func(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	PutTargetsFunc            func(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error)
	DeleteRuleFunc            func(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
	RemoveTargetsFunc         func(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error)
	EBListTagsForResourceFunc func(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error)
	ListRuleNamesByTargetFunc func(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error)
	EBTagResourceFunc         func(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error)
}

type mockClientCallCount struct {
	CreateStateMachine    int
	DescribeStateMachine  int
	DeleteStateMachine    int
	DescribeLogGroups     int
	ListStateMachines     int
	UpdateStateMachine    int
	PutRule               int
	DescribeRule          int
	ListTargetsByRule     int
	PutTargets            int
	DeleteRule            int
	RemoveTargets         int
	ListRuleNamesByTarget int

	SFnListTagsForResource int
	EBListTagsForResource  int
	SFnTagResource         int
	EBTagResource          int
}

func (m *mockClientCallCount) Reset() {
	m.CreateStateMachine = 0
	m.DescribeStateMachine = 0
	m.DeleteStateMachine = 0
	m.DescribeLogGroups = 0
	m.ListStateMachines = 0
	m.UpdateStateMachine = 0
	m.SFnTagResource = 0
	m.EBTagResource = 0
	m.PutRule = 0
	m.DescribeRule = 0
	m.ListTargetsByRule = 0
	m.PutTargets = 0
	m.DeleteRule = 0
	m.RemoveTargets = 0
	m.ListRuleNamesByTarget = 0

	m.SFnListTagsForResource = 0
	m.EBListTagsForResource = 0
}

type mockSFnClient struct {
	stefunny.SFnClient
	aws *mockAWSClient
}

type mockCWLogsClient struct {
	stefunny.CWLogsClient
	aws *mockAWSClient
}

type mockEBClient struct {
	stefunny.EventBridgeClient
	aws *mockAWSClient
}

func (m *mockSFnClient) CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
	m.aws.CallCount.CreateStateMachine++
	if m.aws.CreateStateMachineFunc == nil {
		return nil, errors.New("unexpected Call CreateStateMachine")
	}
	return m.aws.CreateStateMachineFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	m.aws.CallCount.DescribeStateMachine++
	if m.aws.DescribeStateMachineFunc == nil {
		return nil, errors.New("unexpected Call DescribeStateMachine")
	}
	return m.aws.DescribeStateMachineFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error) {
	m.aws.CallCount.DeleteStateMachine++
	if m.aws.DeleteStateMachineFunc == nil {
		return nil, errors.New("unexpected Call DeleteStateMachine")
	}
	return m.aws.DeleteStateMachineFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) ListStateMachines(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
	m.aws.CallCount.ListStateMachines++
	if m.aws.ListStateMachinesFunc == nil {
		return nil, errors.New("unexpected Call ListStateMachines")
	}
	return m.aws.ListStateMachinesFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
	m.aws.CallCount.UpdateStateMachine++
	if m.aws.UpdateStateMachineFunc == nil {
		return nil, errors.New("unexpected Call UpdateStateMachine")
	}
	return m.aws.UpdateStateMachineFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) ListTagsForResource(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error) {
	m.aws.CallCount.SFnListTagsForResource++
	if m.aws.SFnListTagsForResourceFunc == nil {
		return nil, errors.New("unexpected Call ListTagsForResource")
	}
	return m.aws.SFnListTagsForResourceFunc(ctx, params, optFns...)
}

func (m *mockCWLogsClient) DescribeLogGroups(ctx context.Context, params *cloudwatchlogs.DescribeLogGroupsInput, optFns ...func(*cloudwatchlogs.Options)) (*cloudwatchlogs.DescribeLogGroupsOutput, error) {
	m.aws.CallCount.DescribeLogGroups++
	if m.aws.DescribeLogGroupsFunc == nil {
		return nil, errors.New("unexpected Call DescribeLogGroups")
	}
	return m.aws.DescribeLogGroupsFunc(ctx, params, optFns...)
}

func (m *mockSFnClient) TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error) {
	m.aws.CallCount.SFnTagResource++
	if m.aws.SFnTagResourceFunc == nil {
		return nil, errors.New("unexpected Call TagResource")
	}
	return m.aws.SFnTagResourceFunc(ctx, params, optFns...)
}

func (m *mockEBClient) TagResource(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error) {
	m.aws.CallCount.EBTagResource++
	if m.aws.EBTagResourceFunc == nil {
		return nil, errors.New("unexpected Call TagResource")
	}
	return m.aws.EBTagResourceFunc(ctx, params, optFns...)
}

func (m *mockEBClient) PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error) {
	m.aws.CallCount.PutRule++
	if m.aws.PutRuleFunc == nil {
		return nil, errors.New("unexpected Call PutRule")
	}
	return m.aws.PutRuleFunc(ctx, params, optFns...)
}
func (m *mockEBClient) DescribeRule(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error) {
	m.aws.CallCount.DescribeRule++
	if m.aws.DescribeRuleFunc == nil {
		return nil, errors.New("unexpected Call DescribeRule")
	}
	return m.aws.DescribeRuleFunc(ctx, params, optFns...)
}
func (m *mockEBClient) ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error) {
	m.aws.CallCount.ListTargetsByRule++
	if m.aws.ListTargetsByRuleFunc == nil {
		return nil, errors.New("unexpected Call ListTargetsByRule")
	}
	return m.aws.ListTargetsByRuleFunc(ctx, params, optFns...)
}
func (m *mockEBClient) PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error) {
	m.aws.CallCount.PutTargets++
	if m.aws.PutTargetsFunc == nil {
		return nil, errors.New("unexpected Call PutTargets")
	}
	return m.aws.PutTargetsFunc(ctx, params, optFns...)
}
func (m *mockEBClient) DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error) {
	m.aws.CallCount.DeleteRule++
	if m.aws.DeleteRuleFunc == nil {
		return nil, errors.New("unexpected Call DeleteRule")
	}
	return m.aws.DeleteRuleFunc(ctx, params, optFns...)
}
func (m *mockEBClient) RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error) {
	m.aws.CallCount.RemoveTargets++
	if m.aws.RemoveTargetsFunc == nil {
		return nil, errors.New("unexpected Call RemoveTargets")
	}
	return m.aws.RemoveTargetsFunc(ctx, params, optFns...)
}

func (m *mockEBClient) ListTagsForResource(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error) {
	m.aws.CallCount.EBListTagsForResource++
	if m.aws.EBListTagsForResourceFunc == nil {
		return nil, errors.New("unexpected Call ListTagsForResource")
	}
	return m.aws.EBListTagsForResourceFunc(ctx, params, optFns...)
}

func (m *mockEBClient) ListRuleNamesByTarget(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error) {
	m.aws.CallCount.ListRuleNamesByTarget++
	if m.aws.ListRuleNamesByTargetFunc == nil {
		return nil, errors.New("unexpected Call ListTagsForResource")
	}
	return m.aws.ListRuleNamesByTargetFunc(ctx, params, optFns...)
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

func (m *mockAWSClient) Clone() *mockAWSClient {
	ret := *m
	return &ret
}

func (m *mockAWSClient) Overwrite(o *mockAWSClient) *mockAWSClient {
	ret := m.Clone()
	if o.CreateStateMachineFunc != nil {
		ret.CreateStateMachineFunc = o.CreateStateMachineFunc
	}
	if o.DescribeStateMachineFunc != nil {
		ret.DescribeStateMachineFunc = o.DescribeStateMachineFunc
	}
	if o.DeleteStateMachineFunc != nil {
		ret.DeleteStateMachineFunc = o.DeleteStateMachineFunc
	}
	if o.ListStateMachinesFunc != nil {
		ret.ListStateMachinesFunc = o.ListStateMachinesFunc
	}
	if o.UpdateStateMachineFunc != nil {
		ret.UpdateStateMachineFunc = o.UpdateStateMachineFunc
	}
	if o.SFnTagResourceFunc != nil {
		ret.SFnTagResourceFunc = o.SFnTagResourceFunc
	}
	if o.EBTagResourceFunc != nil {
		ret.EBTagResourceFunc = o.EBTagResourceFunc
	}
	if o.DescribeLogGroupsFunc != nil {
		ret.DescribeLogGroupsFunc = o.DescribeLogGroupsFunc
	}
	if o.PutRuleFunc != nil {
		ret.PutRuleFunc = o.PutRuleFunc
	}
	if o.DescribeRuleFunc != nil {
		ret.DescribeRuleFunc = o.DescribeRuleFunc
	}
	if o.ListTargetsByRuleFunc != nil {
		ret.ListTargetsByRuleFunc = o.ListTargetsByRuleFunc
	}
	if o.PutTargetsFunc != nil {
		ret.PutTargetsFunc = o.PutTargetsFunc
	}
	if o.DeleteRuleFunc != nil {
		ret.DeleteRuleFunc = o.DeleteRuleFunc
	}
	if o.RemoveTargetsFunc != nil {
		ret.RemoveTargetsFunc = o.RemoveTargetsFunc
	}
	if o.ListRuleNamesByTargetFunc != nil {
		ret.ListRuleNamesByTargetFunc = o.ListRuleNamesByTargetFunc
	}

	return ret
}

func newStateMachineListItem(name string) sfntypes.StateMachineListItem {
	return sfntypes.StateMachineListItem{
		CreationDate:    aws.Time(time.Date(2021, 10, 1, 2, 3, 4, 5, time.UTC)),
		Name:            aws.String(name),
		StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:%s", name)),
	}
}

func newMockApp(t *testing.T, path string, client *mockAWSClient) *stefunny.App {
	t.Helper()
	l := stefunny.NewConfigLoader(nil, nil)
	ctx := context.Background()
	err := l.AppendTFState(ctx, "", "testdata/terraform.tfstate")
	require.NoError(t, err)
	cfg, err := l.Load(path)
	require.NoError(t, err)
	app, err := stefunny.NewWithClient(cfg, stefunny.AWSClients{
		SFnClient:         &mockSFnClient{aws: client},
		CWLogsClient:      &mockCWLogsClient{aws: client},
		EventBridgeClient: &mockEBClient{aws: client},
	})
	require.NoError(t, err)
	return app
}
