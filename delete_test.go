package stefunny_test

import (
	"context"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {

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
				DeleteStateMachine:   0,
			},
		},
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			DryRun:   false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:    1,
				DescribeStateMachine: 1,
				DeleteStateMachine:   1,
			},
		},
		{
			casename: "deleting",
			path:     "testdata/deleting.yaml",
			DryRun:   false,
			expectedCallCount: mockClientCallCount{
				ListStateMachines:    1,
				DescribeStateMachine: 1,
				DeleteStateMachine:   0,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			client.CallCount.Reset()

			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path, stefunny.LoadConfigOption{
				TFState: "testdata/terraform.tfstate",
			})
			require.NoError(t, err)
			app, err := stefunny.NewWithClient(cfg, stefunny.AWSClients{
				CWLogsClient: client,
				SFnClient:    client,
			})
			require.NoError(t, err)
			err = app.Delete(context.Background(), stefunny.DeleteOption{
				DryRun: c.DryRun,
				Force:  true,
			})
			require.NoError(t, err)
			require.EqualValues(t, c.expectedCallCount, client.CallCount)
		})
	}
}
