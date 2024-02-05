package stefunny_test

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {
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
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.sfn.On("UpdateStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.UpdateStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Hello")
					},
				)).Return(
					&sfn.UpdateStateMachineOutput{
						StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:12345"),
						RevisionId:             aws.String("123456"),
						UpdateDate:             aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("TagResource", mock.Anything, mock.MatchedBy(
					func(input *sfn.TagResourceInput) bool {
						return assert.Contains(t, *input.ResourceArn, "Hello")
					},
				)).Return(
					&sfn.TagResourceOutput{},
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
			casename: "not_found_and_create",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(
					&sfn.ListStateMachinesOutput{StateMachines: []sfntypes.StateMachineListItem{}},
					nil,
				).Once()
				m.sfn.On("CreateStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.CreateStateMachineInput) bool {
						return assert.Contains(t, *input.Name, "Hello")
					},
				)).Return(
					&sfn.CreateStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
						CreationDate:    aws.Time(time.Now()),
					},
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
			casename: "not_found_and_create_with_schedule",
			path:     "testdata/schedule.yaml",
			DryRun:   false,
			setupMocks: func(t *testing.T, m *mocks) {
				m.sfn.On("ListStateMachines", mock.Anything, mock.Anything).Return(
					&sfn.ListStateMachinesOutput{StateMachines: []sfntypes.StateMachineListItem{}},
					nil,
				).Once()
				m.sfn.On("CreateStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.CreateStateMachineInput) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&sfn.CreateStateMachineOutput{
						StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Scheduled"),
						CreationDate:    aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Scheduled")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{}},
					nil,
				).Once()
				m.eventBridge.On("PutRule", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.PutRuleInput) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&eventbridge.PutRuleOutput{RuleArn: aws.String("arn:aws:events:us-east-1:123456789012:rule/Scheduled")},
					nil,
				).Once()
				m.eventBridge.On("PutTargets", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.PutTargetsInput) bool {
						return assert.Contains(t, *input.Rule, "Scheduled")
					},
				)).Return(
					&eventbridge.PutTargetsOutput{},
					nil,
				).Once()
				m.eventBridge.On("TagResource", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.TagResourceInput) bool {
						return assert.Equal(t, *input.ResourceARN, "arn:aws:events:us-east-1:123456789012:rule/Scheduled")
					},
				)).Return(
					&eventbridge.TagResourceOutput{},
					nil,
				).Once()
			},
		},
		{
			casename: "deploy with schedule",
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
				m.sfn.On("ListTagsForResource", mock.Anything, mock.Anything).Return(
					&sfn.ListTagsForResourceOutput{Tags: []sfntypes.Tag{}},
					nil,
				).Once()
				m.sfn.On("UpdateStateMachine", mock.Anything, mock.MatchedBy(
					func(input *sfn.UpdateStateMachineInput) bool {
						return assert.Contains(t, *input.StateMachineArn, "Scheduled")
					},
				)).Return(
					&sfn.UpdateStateMachineOutput{
						StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Scheduled:12345"),
						RevisionId:             aws.String("123456"),
						UpdateDate:             aws.Time(time.Now()),
					},
					nil,
				).Once()
				m.sfn.On("TagResource", mock.Anything, mock.MatchedBy(
					func(input *sfn.TagResourceInput) bool {
						return assert.Contains(t, *input.ResourceArn, "Scheduled")
					},
				)).Return(
					&sfn.TagResourceOutput{},
					nil,
				).Once()
				m.eventBridge.On("ListRuleNamesByTarget", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.ListRuleNamesByTargetInput) bool {
						return assert.NotNil(t, input.TargetArn) &&
							assert.Contains(t, *input.TargetArn, "arn:aws:states:") &&
							assert.Contains(t, *input.TargetArn, "Scheduled")
					},
				)).Return(
					&eventbridge.ListRuleNamesByTargetOutput{RuleNames: []string{}},
					nil,
				).Once()
				m.eventBridge.On("PutRule", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.PutRuleInput) bool {
						return assert.Contains(t, *input.Name, "Scheduled")
					},
				)).Return(
					&eventbridge.PutRuleOutput{RuleArn: aws.String("arn:aws:events:us-east-1:123456789012:rule/Scheduled")},
					nil,
				).Once()
				m.eventBridge.On("PutTargets", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.PutTargetsInput) bool {
						return assert.Contains(t, *input.Rule, "Scheduled")
					},
				)).Return(
					&eventbridge.PutTargetsOutput{},
					nil,
				).Once()
				m.eventBridge.On("TagResource", mock.Anything, mock.MatchedBy(
					func(input *eventbridge.TagResourceInput) bool {
						return assert.Equal(t, *input.ResourceARN, "arn:aws:events:us-east-1:123456789012:rule/Scheduled")
					},
				)).Return(
					&eventbridge.TagResourceOutput{},
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
			err := app.Deploy(context.Background(), stefunny.DeployOption{
				DryRun: c.DryRun,
			})
			require.NoError(t, err)
		})
	}
}
