package stefunny_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDelete(t *testing.T) {

	client := &mockAWSClient{
		ListStateMachinesFunc: func(ctx context.Context, params *sfn.ListStateMachinesInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachinesOutput, error) {
			return &sfn.ListStateMachinesOutput{
				StateMachines: []sfntypes.StateMachineListItem{
					{
						Name:            aws.String("Hello"),
						StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
					},
					{
						Name:            aws.String("Deleting"),
						StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Deleting"),
					},
				},
			}, nil
		},
		DescribeStateMachineFunc: func(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
			status := sfntypes.StateMachineStatusActive
			if strings.HasSuffix(*params.StateMachineArn, "Deleting") {
				status = sfntypes.StateMachineStatusDeleting
			}
			return &sfn.DescribeStateMachineOutput{
				CreationDate:    aws.Time(time.Now()),
				StateMachineArn: params.StateMachineArn,
				Status:          status,
			}, nil
		},
		DeleteStateMachineFunc: func(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error) {
			return &sfn.DeleteStateMachineOutput{}, nil
		},
	}
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
			err := cfg.Load(c.path)
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
