package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
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
	ErrStateMachineDoesNotExist = errors.New("state machine does not exist")
	ErrLogGroupNotFound         = errors.New("log group not found")
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
		if _, ok := err.(*sfntypes.StateMachineDoesNotExist); ok {
			return nil, ErrStateMachineDoesNotExist
		}
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
	return "", ErrStateMachineDoesNotExist
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

type DeployStateMachineOutput struct {
	CreationDate    *time.Time
	UpdateDate      *time.Time
	StateMachineArn *string
}

func (svc *AWSService) DeployStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	if stateMachine.StateMachineArn == nil {
		log.Println("[debug] try create state machine")
		output, err := svc.SFnClient.CreateStateMachine(ctx, &stateMachine.CreateStateMachineInput, optFns...)
		if err != nil {
			return nil, fmt.Errorf("create failed: %w", err)
		}
		log.Println("[debug] finish create state machine")
		return &DeployStateMachineOutput{
			StateMachineArn: output.StateMachineArn,
			CreationDate:    output.CreationDate,
			UpdateDate:      output.CreationDate,
		}, nil
	} else {
		return svc.updateStateMachine(ctx, stateMachine, optFns...)
	}
}

func (svc *AWSService) updateStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	log.Println("[debug] try update state machine")
	output, err := svc.SFnClient.UpdateStateMachine(ctx, &sfn.UpdateStateMachineInput{
		StateMachineArn:      stateMachine.StateMachineArn,
		Definition:           stateMachine.Definition,
		LoggingConfiguration: stateMachine.LoggingConfiguration,
		RoleArn:              stateMachine.RoleArn,
		TracingConfiguration: stateMachine.TracingConfiguration,
	}, optFns...)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] finish update state machine")

	log.Println("[debug] try update state machine tags")
	_, err = svc.SFnClient.TagResource(ctx, &sfn.TagResourceInput{
		ResourceArn: stateMachine.StateMachineArn,
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	log.Println("[debug] finish update state machine tags")
	return &DeployStateMachineOutput{
		StateMachineArn: stateMachine.StateMachineArn,
		CreationDate:    stateMachine.CreationDate,
		UpdateDate:      output.UpdateDate,
	}, nil
}

func (s *StateMachine) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(s.configureJSON())
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(*s.Definition)
	return builder.String()
}

func (s *StateMachine) DiffString(newStateMachine *StateMachine) string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(jsonDiffString(s.configureJSON(), newStateMachine.configureJSON()))
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(jsonDiffString(*s.Definition, *newStateMachine.Definition))
	return builder.String()
}

func (s *StateMachine) configureJSON() string {
	params := map[string]interface{}{
		"Name":                 s.Name,
		"RoleArn":              s.RoleArn,
		"LoggingConfiguration": s.LoggingConfiguration,
		"TracingConfiguration": &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		"Type": s.Type,
	}
	if s.TracingConfiguration != nil {
		params["TracingConfiguration"] = s.TracingConfiguration
	}
	return marshalJSONString(params)
}
