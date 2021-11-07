package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {

	client := getDefaultMock(t)
	cases := []struct {
		client            *mockAWSClient
		casename          string
		path              string
		DryRun            bool
		expectedCallCount mockClientCallCount
	}{
		{
			casename: "default_config dryrun",
			path:     "testdata/default.yaml",
			DryRun:   true,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:      1,
				DescribeStateMachine:   1,
				DescribeLogGroups:      1,
				UpdateStateMachine:     0,
				TagResource:            0,
				DescribeRule:           1,
				SFnListTagsForResource: 1,
			},
		},
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			DryRun:   false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:      1,
				DescribeStateMachine:   1,
				DescribeLogGroups:      1,
				UpdateStateMachine:     1,
				TagResource:            1,
				DescribeRule:           1,
				SFnListTagsForResource: 1,
			},
		},
		{
			casename: "not_found_and_create",
			client: client.Overwrite(&mockAWSClient{
				ListStateMachinesFunc: func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
					return &sfn.ListStateMachinesOutput{
						StateMachines: []sfntypes.StateMachineListItem{},
					}, nil
				},
			}),
			path:   "testdata/default.yaml",
			DryRun: false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:    1,
				DescribeStateMachine: 0,
				DescribeLogGroups:    1,
				CreateStateMachine:   1,
				TagResource:          0,
				DescribeRule:         1,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			if c.client == nil {
				c.client = client
			}
			c.client.CallCount.Reset()
			app := newMockApp(t, c.path, c.client)
			err := app.Deploy(context.Background(), stefunny.DeployOption{
				DryRun: c.DryRun,
			})
			require.NoError(t, err)
			require.EqualValues(t, c.expectedCallCount, c.client.CallCount)
		})
	}
}
