package stefunny_test

import (
	"context"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDeploy(t *testing.T) {

	client := getDefaultMock(t)
	cases := []struct {
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
				ListStateMachines:    1,
				DescribeStateMachine: 1,
				DescribeLogGroups:    1,
				UpdateStateMachine:   0,
			},
		},
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			DryRun:   false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:    1,
				DescribeStateMachine: 1,
				DescribeLogGroups:    1,
				UpdateStateMachine:   1,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			client.CallCount.Reset()

			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			app, err := stefunny.NewWithClient(cfg, stefunny.AWSClients{
				CWLogsClient: client,
				SFnClient:    client,
			})
			require.NoError(t, err)
			err = app.Deploy(context.Background(), stefunny.DeployOption{
				DryRun: c.DryRun,
			})
			require.NoError(t, err)
			require.EqualValues(t, c.expectedCallCount, client.CallCount)
		})
	}
}
