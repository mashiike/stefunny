package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/smithy-go"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestEventBridgeService__SearchRealtedRules(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockEventBridgeClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListRuleNamesByTarget(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListRuleNamesByTargetInput) bool {
			return input.TargetArn != nil && *input.TargetArn == "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"
		},
	)).Return(
		&eventbridge.ListRuleNamesByTargetOutput{
			RuleNames: []string{"Scheduled"},
		},
		nil,
	).Times(1)
	m.EXPECT().ListRuleNamesByTarget(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListRuleNamesByTargetInput) bool {
			return input.TargetArn != nil && *input.TargetArn == "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"
		},
	)).Return(
		&eventbridge.ListRuleNamesByTargetOutput{
			RuleNames: []string{"Unqualified"},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeRule(gomock.Any(), &eventbridge.DescribeRuleInput{
		Name: aws.String("Scheduled"),
	}).Return(
		&eventbridge.DescribeRuleOutput{
			Name:         aws.String("Scheduled"),
			State:        eventbridgetypes.RuleStateDisabled,
			Arn:          aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
			RoleArn:      aws.String("arn:aws:iam::000000000000:role/service-role/StatesExecutionRole-us-east-1"),
			EventBusName: aws.String("default"),
		},
		nil,
	).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &eventbridge.ListTagsForResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
	}).Return(
		&eventbridge.ListTagsForResourceOutput{
			Tags: []eventbridgetypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeRule(gomock.Any(), &eventbridge.DescribeRuleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&eventbridge.DescribeRuleOutput{
			Name:         aws.String("Unqualified"),
			State:        eventbridgetypes.RuleStateEnabled,
			Arn:          aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
			EventBusName: aws.String("default"),
		},
		nil,
	).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &eventbridge.ListTagsForResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
	}).Return(
		&eventbridge.ListTagsForResourceOutput{
			Tags: []eventbridgetypes.Tag{},
		},
		nil,
	).Times(1)
	m.EXPECT().ListTargetsByRule(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListTargetsByRuleInput) bool {
			return *input.Rule == "Scheduled"
		},
	)).Return(
		&eventbridge.ListTargetsByRuleOutput{
			Targets: []eventbridgetypes.Target{
				{
					Id:  aws.String("stefunny-managed"),
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListTargetsByRule(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListTargetsByRuleInput) bool {
			return *input.Rule == "Unqualified"
		},
	)).Return(
		&eventbridge.ListTargetsByRuleOutput{
			Targets: []eventbridgetypes.Target{
				{
					Id:  aws.String("Id0000000-0000-0000-0000-000000000000"),
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
				},
				{
					Id:  aws.String("Id0000000-0000-0000-0000-000000000001"),
					Arn: aws.String("arn:aws:lambda:us-east-1:000000000000:function:Unqualified"),
				},
			},
		},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewEventBridgeService(m)
	rules, err := svc.SearchRelatedRules(ctx, &stefunny.SearchRelatedRulesInput{
		StateMachineQualifiedArn: "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
		RuleNames:                []string{"Scheduled"},
	})
	require.NoError(t, err)
	require.EqualValues(t, stefunny.EventBridgeRules{
		{
			PutRuleInput: eventbridge.PutRuleInput{
				Name:    aws.String("Scheduled"),
				State:   eventbridgetypes.RuleStateDisabled,
				RoleArn: aws.String("arn:aws:iam::000000000000:role/service-role/StatesExecutionRole-us-east-1"),
				Tags: []eventbridgetypes.Tag{
					{
						Key:   aws.String("ManagedBy"),
						Value: aws.String("stefunny"),
					},
				},
				EventBusName: aws.String("default"),
			},
			RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
			Target: eventbridgetypes.Target{
				Id:  aws.String("stefunny-managed"),
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
			},
			AdditionalTargets: []eventbridgetypes.Target{},
		},
		{
			PutRuleInput: eventbridge.PutRuleInput{
				Name:         aws.String("Unqualified"),
				State:        eventbridgetypes.RuleStateEnabled,
				EventBusName: aws.String("default"),
				Tags:         []eventbridgetypes.Tag{},
			},
			RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
			Target: eventbridgetypes.Target{
				Id:  aws.String("Id0000000-0000-0000-0000-000000000000"),
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
			},
			AdditionalTargets: []eventbridgetypes.Target{
				{
					Id:  aws.String("Id0000000-0000-0000-0000-000000000001"),
					Arn: aws.String("arn:aws:lambda:us-east-1:000000000000:function:Unqualified"),
				},
			},
		},
	}, rules)
}

func TestEventBridgeService__DeployRules(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockEventBridgeClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListRuleNamesByTarget(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListRuleNamesByTargetInput) bool {
			return input.TargetArn != nil && *input.TargetArn == "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"
		},
	)).Return(
		&eventbridge.ListRuleNamesByTargetOutput{
			RuleNames: []string{"Scheduled"},
		},
		nil,
	).Times(1)
	m.EXPECT().ListRuleNamesByTarget(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListRuleNamesByTargetInput) bool {
			return input.TargetArn != nil && *input.TargetArn == "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"
		},
	)).Return(
		&eventbridge.ListRuleNamesByTargetOutput{
			RuleNames: []string{"Unqualified"},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeRule(gomock.Any(), &eventbridge.DescribeRuleInput{
		Name: aws.String("Scheduled"),
	}).Return(
		&eventbridge.DescribeRuleOutput{
			Name:         aws.String("Scheduled"),
			State:        eventbridgetypes.RuleStateDisabled,
			Arn:          aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
			RoleArn:      aws.String("arn:aws:iam::000000000000:role/service-role/StatesExecutionRole-us-east-1"),
			EventBusName: aws.String("default"),
		},
		nil,
	).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &eventbridge.ListTagsForResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
	}).Return(
		&eventbridge.ListTagsForResourceOutput{
			Tags: []eventbridgetypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeRule(gomock.Any(), &eventbridge.DescribeRuleInput{
		Name: aws.String("Unqualified"),
	}).Return(
		&eventbridge.DescribeRuleOutput{
			Name:         aws.String("Unqualified"),
			State:        eventbridgetypes.RuleStateEnabled,
			Arn:          aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
			EventBusName: aws.String("default"),
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeRule(gomock.Any(), &eventbridge.DescribeRuleInput{
		Name: aws.String("Event"),
	}).Return(
		nil,
		&smithy.GenericAPIError{Code: "ResourceNotFoundException"},
	).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &eventbridge.ListTagsForResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
	}).Return(
		&eventbridge.ListTagsForResourceOutput{
			Tags: []eventbridgetypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListTargetsByRule(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListTargetsByRuleInput) bool {
			return *input.Rule == "Scheduled"
		},
	)).Return(
		&eventbridge.ListTargetsByRuleOutput{
			Targets: []eventbridgetypes.Target{
				{
					Id:  aws.String("stefunny-managed"),
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListTargetsByRule(gomock.Any(), gomock.Cond(
		func(input *eventbridge.ListTargetsByRuleInput) bool {
			return *input.Rule == "Unqualified"
		},
	)).Return(
		&eventbridge.ListTargetsByRuleOutput{
			Targets: []eventbridgetypes.Target{
				{
					Id:  aws.String("Id0000000-0000-0000-0000-000000000000"),
					Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
				},
				{
					Id:  aws.String("Id0000000-0000-0000-0000-000000000001"),
					Arn: aws.String("arn:aws:lambda:us-east-1:000000000000:function:Unqualified"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().RemoveTargets(gomock.Any(), &eventbridge.RemoveTargetsInput{
		Rule:         aws.String("Unqualified"),
		Ids:          []string{"Id0000000-0000-0000-0000-000000000000", "Id0000000-0000-0000-0000-000000000001"},
		EventBusName: aws.String("default"),
	}).Return(
		&eventbridge.RemoveTargetsOutput{},
		nil,
	).Times(1)

	m.EXPECT().DeleteRule(gomock.Any(), gomock.Cond(
		func(input *eventbridge.DeleteRuleInput) bool {
			return *input.Name == "Unqualified"
		},
	)).Return(
		&eventbridge.DeleteRuleOutput{},
		nil,
	).Times(1)

	m.EXPECT().PutRule(gomock.Any(), &eventbridge.PutRuleInput{
		Name:         aws.String("Scheduled"),
		State:        eventbridgetypes.RuleStateDisabled,
		EventBusName: aws.String("default"),
		RoleArn:      aws.String("arn:aws:iam::000000000000:role/service-role/StatesExecutionRole-us-east-1"),
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
		},
	}).Return(
		&eventbridge.PutRuleOutput{
			RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
		},
		nil,
	).Times(1)
	m.EXPECT().PutTargets(gomock.Any(), &eventbridge.PutTargetsInput{
		Rule: aws.String("Scheduled"),
		Targets: []eventbridgetypes.Target{
			{
				Id:  aws.String("stefunny-managed"),
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
			},
		},
	}).Return(
		&eventbridge.PutTargetsOutput{},
		nil,
	).Times(1)
	m.EXPECT().TagResource(gomock.Any(), &eventbridge.TagResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
		},
	}).Return(
		&eventbridge.TagResourceOutput{},
		nil,
	).Times(1)
	m.EXPECT().PutRule(gomock.Any(), &eventbridge.PutRuleInput{
		Name:         aws.String("Event"),
		State:        eventbridgetypes.RuleStateEnabled,
		EventPattern: aws.String(`{"source":["stefunny"],"detail-type":["Scheduled"]}`),
		EventBusName: aws.String("default"),
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
		},
	}).Return(
		&eventbridge.PutRuleOutput{
			RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
		},
		nil,
	).Times(1)
	m.EXPECT().PutTargets(gomock.Any(), &eventbridge.PutTargetsInput{
		Rule: aws.String("Event"),
		Targets: []eventbridgetypes.Target{
			{
				Id:  aws.String("stefunny-managed"),
				Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current"),
			},
		},
	}).Return(
		&eventbridge.PutTargetsOutput{},
		nil,
	).Times(1)
	m.EXPECT().TagResource(gomock.Any(), &eventbridge.TagResourceInput{
		ResourceARN: aws.String("arn:aws:events:us-east-1:000000000000:rule/Unqualified"),
		Tags: []eventbridgetypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
		},
	}).Return(
		&eventbridge.TagResourceOutput{},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewEventBridgeService(m)
	err := svc.DeployRules(ctx, "arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled:current",
		stefunny.EventBridgeRules{
			{
				PutRuleInput: eventbridge.PutRuleInput{
					Name:         aws.String("Scheduled"),
					State:        eventbridgetypes.RuleStateEnabled,
					RoleArn:      aws.String("arn:aws:iam::000000000000:role/service-role/StatesExecutionRole-us-east-1"),
					EventBusName: aws.String("default"),
				},
				RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
				Target: eventbridgetypes.Target{
					Id: aws.String("stefunny-managed"),
				},
				AdditionalTargets: []eventbridgetypes.Target{},
			},
			{
				PutRuleInput: eventbridge.PutRuleInput{
					Name:         aws.String("Event"),
					State:        eventbridgetypes.RuleStateEnabled,
					EventBusName: aws.String("default"),
					EventPattern: aws.String(`{"source":["stefunny"],"detail-type":["Scheduled"]}`),
				},
				RuleArn: aws.String("arn:aws:events:us-east-1:000000000000:rule/Event"),
				Target: eventbridgetypes.Target{
					Id: aws.String("stefunny-managed"),
				},
				AdditionalTargets: []eventbridgetypes.Target{},
			},
		},
		true,
	)
	require.NoError(t, err)
}
