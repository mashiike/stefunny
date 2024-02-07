package stefunny_test

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cloudwatchlogstypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadValid(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	cases := []struct {
		casename    string
		path        string
		expectedDef string
		extStr      map[string]string
		extCode     map[string]string
		setupLoader func(t *testing.T, l *stefunny.ConfigLoader)
	}{
		{
			casename:    "default_config",
			path:        "testdata/stefunny.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "yaml_config_with_jsonnet_def",
			path:        "testdata/jsonnet_def.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "log_level_off",
			path:        "testdata/logging_off.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename: "tfstate_read",
			path:     "testdata/tfstate.yaml",
			extStr: map[string]string{
				"Comment": "great!!!",
			},
			extCode: map[string]string{
				"WaitSeconds": "60*2",
			},
			expectedDef: LoadString(t, "testdata/tfstate.asl.json"),
		},
		{
			casename:    "yaml",
			path:        "testdata/yaml_def.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "jsonnet",
			path:        "testdata/stefunny.jsonnet",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "schedule",
			path:        "testdata/schedule.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "old_type_config_v0.5.0",
			path:        "testdata/old_config.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
			setupLoader: func(t *testing.T, l *stefunny.ConfigLoader) {
				client := NewMockCloudWatchLogsClient(t)
				client.On("DescribeLogGroups", mock.Anything, mock.Anything).Return(&cloudwatchlogs.DescribeLogGroupsOutput{
					LogGroups: []cloudwatchlogstypes.LogGroup{
						{
							LogGroupName: aws.String("/aws/vendedlogs/states/Hello-Logs"),
							Arn:          aws.String("arn:aws:logs:us-east-1:000000000000:log-group:/aws/vendedlogs/states/Hello-Logs"),
						},
					},
				}, nil).Once()
				l.SetCloudWatchLogsClient(client)
			},
		},
	}
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/config"),
		goldie.WithNameSuffix(".golden.json"),
	)
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			l := stefunny.NewConfigLoader(c.extStr, c.extCode)
			if c.setupLoader != nil {
				c.setupLoader(t, l)
			}
			ctx := context.Background()
			cfg, err := l.Load(ctx, c.path)
			require.NoError(t, err)
			require.NotNil(t, cfg.StateMachine.Value.Definition)
			require.JSONEq(t, c.expectedDef, *cfg.StateMachine.Value.Definition)
			g.AssertJson(t, c.casename, cfg)
		})
	}

}

func TestConfigLoadInValid(t *testing.T) {
	cases := []struct {
		casename string
		path     string
		expected string
	}{
		{
			casename: "no_such_file",
			path:     "testdata/not_found.yaml",
		},
		{
			casename: "level_invalid",
			path:     "testdata/hoge_level.yaml",
			expected: "state_machine.logging_configuration.level is invalid level: please ALL, ERROR, FATAL, or OFF",
		},
		{
			casename: "type_invalid",
			path:     "testdata/hoge_type.yaml",
			expected: "state_machine.type is invalid type: please STANDARD, EXPRESS",
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			t.Log("test location:", dataloc.L(c.casename))
			l := stefunny.NewConfigLoader(nil, nil)
			ctx := context.Background()
			_, err := l.Load(ctx, c.path)
			require.Error(t, err)
			if c.expected != "" {
				require.ErrorContains(t, err, c.expected)
			}
		})
	}

}
