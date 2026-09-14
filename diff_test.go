package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDiff(t *testing.T) {
	const stateMachineArn = "arn:aws:states:us-east-1:000000000000:stateMachine:Hello"
	const qualifiedArn = stateMachineArn + ":current"

	cases := []struct {
		casename        string
		exitCode        bool
		roleArn         string
		orphanRule      bool
		skipTrigger     bool
		liveOnlyTag     bool
		tagStrategy     *string
		managedByTagKey string
		wantErrHasDiff  bool
	}{
		{
			casename:       "no diff, exit-code on",
			exitCode:       true,
			wantErrHasDiff: false,
		},
		{
			casename:       "live-only tag, default strategy (append_only), exit-code on",
			exitCode:       true,
			liveOnlyTag:    true,
			wantErrHasDiff: false,
		},
		{
			casename:       "live-only tag, sync, exit-code on",
			exitCode:       true,
			liveOnlyTag:    true,
			tagStrategy:    aws.String("sync"),
			wantErrHasDiff: true,
		},
		{
			casename:       "state machine diff, exit-code on",
			exitCode:       true,
			roleArn:        "arn:aws:iam::999999999999:role/other-role",
			wantErrHasDiff: true,
		},
		{
			casename:       "state machine diff, exit-code off (existing behavior unchanged)",
			exitCode:       false,
			roleArn:        "arn:aws:iam::999999999999:role/other-role",
			wantErrHasDiff: false,
		},
		{
			casename:       "eventbridge rule diff only, exit-code on",
			exitCode:       true,
			orphanRule:     true,
			wantErrHasDiff: true,
		},
		{
			casename:       "eventbridge rule diff, skip-trigger on, exit-code on",
			exitCode:       true,
			orphanRule:     true,
			skipTrigger:    true,
			wantErrHasDiff: false,
		},
		{
			casename:       "state machine diff, skip-trigger on, exit-code on",
			exitCode:       true,
			roleArn:        "arn:aws:iam::999999999999:role/other-role",
			skipTrigger:    true,
			wantErrHasDiff: true,
		},
		{
			casename:        "custom managed-by-tag-key, no diff, exit-code on",
			exitCode:        true,
			managedByTagKey: "Owner",
			wantErrHasDiff:  false,
		},
		{
			casename:        "custom managed-by-tag-key, orphan rule tagged with the custom key, exit-code on",
			exitCode:        true,
			orphanRule:      true,
			managedByTagKey: "Owner",
			wantErrHasDiff:  true,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			ctx := context.Background()

			l := stefunny.NewConfigLoader(nil, nil)
			cfg, err := l.Load(ctx, "testdata/stefunny.yaml")
			require.NoError(t, err)
			if c.managedByTagKey != "" {
				cfg.SetManagedByTagKey(c.managedByTagKey)
			}
			newSM := cfg.NewStateMachine()

			current := &stefunny.StateMachine{
				CreateStateMachineInput: newSM.CreateStateMachineInput,
				StateMachineArn:         aws.String(stateMachineArn),
				Status:                  sfntypes.StateMachineStatusActive,
			}
			if c.roleArn != "" {
				current.RoleArn = aws.String(c.roleArn)
			}
			if c.liveOnlyTag {
				current.Tags = append(append([]sfntypes.Tag{}, current.Tags...), sfntypes.Tag{
					Key:   aws.String("Terraform"),
					Value: aws.String("owned"),
				})
			}

			currentRules := stefunny.EventBridgeRules{}
			if c.orphanRule {
				managedByTagKey := "ManagedBy"
				if c.managedByTagKey != "" {
					managedByTagKey = c.managedByTagKey
				}
				currentRules = stefunny.EventBridgeRules{
					{
						PutRuleInput: eventbridge.PutRuleInput{
							Name: aws.String("Hello-orphan"),
							Tags: []eventbridgetypes.Tag{
								{Key: aws.String(managedByTagKey), Value: aws.String("stefunny")},
							},
						},
					},
				}
			}

			mocks := NewMocks(t)
			defer mocks.Finish()
			mocks.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
				Name: "Hello",
			}).Return(current, nil).Times(1)
			if !c.skipTrigger {
				mocks.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), &stefunny.SearchRelatedRulesInput{
					StateMachineQualifiedArn: qualifiedArn,
					RuleNames:                []string{},
				}).Return(currentRules, nil).Times(1)
				mocks.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), &stefunny.SearchRelatedSchedulesInput{
					StateMachineQualifiedArn: qualifiedArn,
					ScheduleNames:            []string{},
				}).Return(stefunny.Schedules{}, nil).Times(1)
			}

			app := newMockApp(t, "testdata/stefunny.yaml", mocks)
			if c.managedByTagKey != "" {
				mocks.sfn.EXPECT().SetManagedByTagKey(c.managedByTagKey).Return()
				mocks.eventBridge.EXPECT().SetManagedByTagKey(c.managedByTagKey).Return()
				app.SetManagedByTagKey(c.managedByTagKey)
			}
			err = app.Diff(ctx, stefunny.DiffOption{
				Unified:     true,
				ExitCode:    c.exitCode,
				SkipTrigger: c.skipTrigger,
				TagStrategy: c.tagStrategy,
			})
			if c.wantErrHasDiff {
				require.ErrorIs(t, err, stefunny.ErrHasDiff)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDiff_QualifierStateMachineNotFound(t *testing.T) {
	LoggerSetup(t, "debug")
	ctx := context.Background()

	mocks := NewMocks(t)
	defer mocks.Finish()
	mocks.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
		Name:      "Hello",
		Qualifier: "not-exist-qualifier",
	}).Return(nil, stefunny.ErrStateMachineDoesNotExist).Times(1)
	mocks.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	}).Return(nil, stefunny.ErrStateMachineDoesNotExist).Times(1)

	app := newMockApp(t, "testdata/stefunny.yaml", mocks)
	var err error
	require.NotPanics(t, func() {
		err = app.Diff(ctx, stefunny.DiffOption{
			Unified:   true,
			Qualifier: "not-exist-qualifier",
		})
	})
	require.NoError(t, err)
}
