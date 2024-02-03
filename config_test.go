package stefunny_test

import (
	"context"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadValid(t *testing.T) {
	cases := []struct {
		casename    string
		path        string
		expectedDef string
		extStr      map[string]string
		extCode     map[string]string
	}{
		{
			casename:    "default_config",
			path:        "testdata/default.yaml",
			expectedDef: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "jsonnet_config",
			path:        "testdata/jsonnet.yaml",
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
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			l := stefunny.NewConfigLoader(c.extStr, c.extCode)
			ctx := context.Background()
			err := l.AppendTFState(ctx, "", "testdata/terraform.tfstate")
			require.NoError(t, err)
			cfg, err := l.Load(c.path)
			require.NoError(t, err)
			require.JSONEq(t, c.expectedDef, cfg.StateMachine.Definition)
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
			expected: "state_machine.logging.level is invalid level: please ALL, ERROR, FATAL, or OFF",
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
			l := stefunny.NewConfigLoader(nil, nil)
			ctx := context.Background()
			err := l.AppendTFState(ctx, "", "testdata/terraform.tfstate")
			require.NoError(t, err)
			_, err = l.Load(c.path)
			require.Error(t, err)
			if c.expected != "" {
				require.ErrorContains(t, err, c.expected)
			}
		})
	}

}
