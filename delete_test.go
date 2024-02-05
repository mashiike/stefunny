package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {
	cases := []struct {
		casename   string
		path       string
		DryRun     bool
		setupMocks func(*testing.T, *mocks)
	}{
		{
			casename: "default_config dryrun",
			path:     "testdata/stefunny.yaml",
			DryRun:   true,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(newListStateMachinesOutput(), nil).Once()
				m.sfn.On("DescribeStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DescribeStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					newDescribeStateMachineOutput("Hello", false),
					nil,
				).Once()
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Hello")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{}},
					nil,
				).Once()
			},
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(newListStateMachinesOutput(), nil).Once()
				m.sfn.On("DescribeStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DescribeStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					newDescribeStateMachineOutput("Hello", false),
					nil,
				).Once()
				m.sfn.On("DeleteStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DeleteStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					&sfn.DeleteStateMachineOutput{},
					nil,
				).Once()
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Hello")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{}},
					nil,
				).Once()
			},
		},
		{
			casename: "deleting",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(newListStateMachinesOutput(), nil).Once()
				m.sfn.On("DescribeStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DescribeStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					newDescribeStateMachineOutput("Hello", true),
					nil,
				).Once()
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Hello")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{}},
					nil,
				).Once()
			},
		},
		{
			casename: "scheduled",
			path:     "testdata/schedule.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(newListStateMachinesOutput(), nil).Once()
				m.sfn.On("DescribeStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DescribeStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Scheduled")
					},
				)).Return(
					newDescribeStateMachineOutput("Scheduled", false),
					nil,
				).Once()
				m.sfn.On("DeleteStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.DeleteStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Scheduled")
					},
				)).Return(
					&sfn.DeleteStateMachineOutput{},
					nil,
				).Once()
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Scheduled")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{"Scheduled"}},
					nil,
				).Once()
				m.eventBridge.On("DescribeRule", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.DescribeRuleInput) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&eventbridge.DescribeRuleOutput{
						Name:               aws.String("Scheduled"),
						Arn:                aws.String("arn:aws:events:us-east-1:000000000000:rule/Scheduled"),
						ScheduleExpression: aws.String("rate(1 hour)"),
						CreatedBy:          aws.String("000000000000"),
					},
					nil,
				).Once()
				m.eventBridge.On("ListTargetsByRule", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListTargetsByRuleInput) bool {
						return assert.Contains(t, *input.Rule, "Scheduled")
					},
				)).Return(
					&eventbridge.ListTargetsByRuleOutput{
						Targets: []eventbridgetypes.Target{
							{
								Id:  aws.String("Scheduled"),
								Arn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Scheduled"),
							},
						},
					},
					nil,
				).Once()
				m.eventBridge.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&eventbridge.ListTagsForResourceOutput{Tags: []eventbridgetypes.Tag{
						{
							Key:   aws.String("ManagedBy"),
							Value: aws.String("stefunny"),
						},
					}},
					nil,
				).Once()
				m.eventBridge.On("RemoveTargets", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.RemoveTargetsInput) bool {
						return assert.Contains(t, *input.Rule, "Scheduled") &&
							assert.Len(t, input.Ids, 1)
					},
				)).Return(
					&eventbridge.RemoveTargetsOutput{},
					nil,
				).Once()
				m.eventBridge.On("DeleteRule", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.DeleteRuleInput) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&eventbridge.DeleteRuleOutput{},
					nil,
				).Once()
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			mocks := NewMocks(t)
			defer mocks.AssertExpectations(t)
			if c.setupMocks != nil {
				c.setupMocks(t, mocks)
			}
			app := newMockApp(t, c.path, mocks)
			err := app.Delete(context.Background(), stefunny.DeleteOption{
				DryRun: c.DryRun,
				Force:  true,
			})
			require.NoError(t, err)
		})
	}
}
