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
		casename       string
		exitCode       bool
		roleArn        string
		orphanRule     bool
		skipTrigger    bool
		wantErrHasDiff bool
	}{
		{
			casename:       "no diff, exit-code on",
			exitCode:       true,
			wantErrHasDiff: false,
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
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			ctx := context.Background()

			l := stefunny.NewConfigLoader(nil, nil)
			cfg, err := l.Load(ctx, "testdata/stefunny.yaml")
			require.NoError(t, err)
			newSM := cfg.NewStateMachine()

			current := &stefunny.StateMachine{
				CreateStateMachineInput: newSM.CreateStateMachineInput,
				StateMachineArn:         aws.String(stateMachineArn),
				Status:                  sfntypes.StateMachineStatusActive,
			}
			if c.roleArn != "" {
				current.RoleArn = aws.String(c.roleArn)
			}

			currentRules := stefunny.EventBridgeRules{}
			if c.orphanRule {
				currentRules = stefunny.EventBridgeRules{
					{
						PutRuleInput: eventbridge.PutRuleInput{
							Name: aws.String("Hello-orphan"),
							Tags: []eventbridgetypes.Tag{
								{Key: aws.String("ManagedBy"), Value: aws.String("stefunny")},
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
			err = app.Diff(ctx, stefunny.DiffOption{
				Unified:     true,
				ExitCode:    c.exitCode,
				SkipTrigger: c.skipTrigger,
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
