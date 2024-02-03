package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
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
			path:     "testdata/stefunny.yaml",
			DryRun:   true,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:      1,
				DescribeStateMachine:   1,
				DescribeLogGroups:      0,
				UpdateStateMachine:     0,
				SFnTagResource:         0,
				DescribeRule:           0,
				SFnListTagsForResource: 1,
				ListRuleNamesByTarget:  1,
			},
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			DryRun:   false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:      1,
				DescribeStateMachine:   1,
				DescribeLogGroups:      0,
				UpdateStateMachine:     1,
				SFnTagResource:         1,
				DescribeRule:           0,
				SFnListTagsForResource: 1,
				ListRuleNamesByTarget:  1,
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
			path:   "testdata/stefunny.yaml",
			DryRun: false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:     1,
				DescribeStateMachine:  0,
				DescribeLogGroups:     0,
				CreateStateMachine:    1,
				SFnTagResource:        0,
				EBTagResource:         0,
				DescribeRule:          0,
				ListRuleNamesByTarget: 1,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
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
