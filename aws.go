package stefunny

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

type SFnClient interface {
	sfn.ListStateMachinesAPIClient
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)
	DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error)
}

type SFnService struct {
	SFnClient
	cacheArnByName map[string]string
}

type CWLogsClient interface {
	cloudwatchlogs.DescribeLogGroupsAPIClient
}

type CWLogsService struct {
	CWLogsClient
	cacheArnByGroupName map[string]string
}

type EventBridgeClient interface {
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
}

type EventBridgeService struct {
	EventBridgeClient
}

func NewSFnService(client SFnClient) *SFnService {
	return &SFnService{
		SFnClient:      client,
		cacheArnByName: make(map[string]string),
	}
}
func NewCWLogsService(client CWLogsClient) *CWLogsService {
	return &CWLogsService{
		CWLogsClient:        client,
		cacheArnByGroupName: make(map[string]string),
	}
}

func NewEventBridgeService(client EventBridgeClient) *EventBridgeService {
	return &EventBridgeService{
		EventBridgeClient: client,
	}
}

var (
	ErrStateMachineNotFound = errors.New("state machine not found")
	ErrLogGroupNotFound     = errors.New("log group not found")
)

func (svc *SFnService) DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	arn, err := svc.GetStateMachineArn(ctx, name, optFns...)
	if err != nil {
		return nil, err
	}
	return svc.SFnClient.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(arn),
	}, optFns...)
}

func (svc *SFnService) GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error) {
	if arn, ok := svc.cacheArnByName[name]; ok {
		return arn, nil
	}
	p := sfn.NewListStateMachinesPaginator(svc.SFnClient, &sfn.ListStateMachinesInput{
		MaxResults: 32,
	})
	for p.HasMorePages() {
		output, err := p.NextPage(ctx, optFns...)
		if err != nil {
			return "", err
		}
		for _, m := range output.StateMachines {
			if *m.Name == name {
				svc.cacheArnByName[name] = *m.StateMachineArn
				return svc.cacheArnByName[name], nil
			}
		}
	}
	return "", ErrStateMachineNotFound
}

func (svc *CWLogsService) GetLogGroupArn(ctx context.Context, name string, optFns ...func(*cloudwatchlogs.Options)) (string, error) {
	if arn, ok := svc.cacheArnByGroupName[name]; ok {
		return arn, nil
	}
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc.CWLogsClient, &cloudwatchlogs.DescribeLogGroupsInput{
		Limit:              aws.Int32(32),
		LogGroupNamePrefix: &name,
	})
	for p.HasMorePages() {
		output, err := p.NextPage(ctx, optFns...)
		if err != nil {
			return "", err
		}
		for _, lg := range output.LogGroups {
			if *lg.LogGroupName == name {
				svc.cacheArnByGroupName[name] = *lg.Arn
				return svc.cacheArnByGroupName[name], nil
			}
		}
	}
	return "", ErrLogGroupNotFound
}
