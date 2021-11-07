package stefunny

import (
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type SFnClient interface {
	sfn.ListStateMachinesAPIClient
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)
	DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error)
}

type CWLogsClient interface {
	cloudwatchlogs.DescribeLogGroupsAPIClient
}
type EventBridgeClient interface {
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	DescribeRule(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
}

type AWSClients struct {
	SFnClient
	CWLogsClient
	EventBridgeClient
}
type AWSService struct {
	SFnClient
	CWLogsClient
	EventBridgeClient
	cacheArnByGroupName        map[string]string
	cacheStateMachineArnByName map[string]string
}

func NewAWSService(clients AWSClients) *AWSService {
	return &AWSService{
		SFnClient:           clients.SFnClient,
		CWLogsClient:        clients.CWLogsClient,
		EventBridgeClient:   clients.EventBridgeClient,
		cacheArnByGroupName: make(map[string]string),

		cacheStateMachineArnByName: make(map[string]string),
	}
}

var (
	ErrLogGroupNotFound = errors.New("log group not found")
)

type StateMachine struct {
	sfn.CreateStateMachineInput
	CreationDate    *time.Time
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
}

func (svc *AWSService) DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*StateMachine, error) {
	arn, err := svc.GetStateMachineArn(ctx, name, optFns...)
	if err != nil {
		return nil, err
	}
	output, err := svc.SFnClient.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(arn),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	stateMachine := &StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Definition:           output.Definition,
			Name:                 output.Name,
			RoleArn:              output.RoleArn,
			LoggingConfiguration: output.LoggingConfiguration,
			TracingConfiguration: output.TracingConfiguration,
			Type:                 output.Type,
		},
		CreationDate:    output.CreationDate,
		StateMachineArn: output.StateMachineArn,
		Status:          output.Status,
	}
	return stateMachine, nil
}

func (svc *AWSService) GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error) {
	if arn, ok := svc.cacheStateMachineArnByName[name]; ok {
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
				svc.cacheStateMachineArnByName[name] = *m.StateMachineArn
				return svc.cacheStateMachineArnByName[name], nil
			}
		}
	}
	return "", &sfntypes.StateMachineDoesNotExist{
		Message: aws.String("ARN could not be searched by Name"),
	}
}

func (svc *AWSService) GetLogGroupArn(ctx context.Context, name string, optFns ...func(*cloudwatchlogs.Options)) (string, error) {
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
