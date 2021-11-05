package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	sfnClient := &mockSFnClient{
		CreateStateMachineFunc: func(_ context.Context, _ *sfn.CreateStateMachineInput, _ ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
			return &sfn.CreateStateMachineOutput{
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
			}, nil
		},
	}
	cwlogsClient := &mockCWLogsClient{
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
	}
	cases := []struct {
		casename                            string
		path                                string
		DryRun                              bool
		expectedCreateStateMachineCallCount int
		expectedDescribeLogGroupsCallCount  int
	}{
		{
			casename:                            "default_config dryrun",
			path:                                "testdata/default.yaml",
			DryRun:                              true,
			expectedCreateStateMachineCallCount: 0,
			expectedDescribeLogGroupsCallCount:  1,
		},
		{
			casename:                            "default_config",
			path:                                "testdata/default.yaml",
			DryRun:                              false,
			expectedCreateStateMachineCallCount: 1,
			expectedDescribeLogGroupsCallCount:  1,
		},
		{
			casename:                            "logging off dryrun",
			path:                                "testdata/full_def.yaml",
			DryRun:                              true,
			expectedCreateStateMachineCallCount: 0,
			expectedDescribeLogGroupsCallCount:  0,
		},
		{
			casename:                            "logging off ",
			path:                                "testdata/full_def.yaml",
			DryRun:                              false,
			expectedCreateStateMachineCallCount: 1,
			expectedDescribeLogGroupsCallCount:  0,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			sfnClient.ResetCallCount()
			cwlogsClient.ResetCallCount()

			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			app, err := stefunny.NewWithClient(cfg, sfnClient, cwlogsClient)
			require.NoError(t, err)
			err = app.Create(context.Background(), stefunny.CreateOption{
				DryRun: c.DryRun,
			})
			require.NoError(t, err)
			require.EqualValues(t, c.expectedCreateStateMachineCallCount, sfnClient.CreateStateMachineCallCount)
			require.EqualValues(t, c.expectedDescribeLogGroupsCallCount, cwlogsClient.DescribeLogGroupsCallCount)
		})
	}
}
