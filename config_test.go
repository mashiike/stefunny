package stefunny_test

import (
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutils"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadValid(t *testing.T) {
	cases := []struct {
		casename    string
		path        string
		expectedDef string
		isYaml      bool
	}{
		{
			casename:    "default_config",
			path:        "testdata/default.yaml",
			expectedDef: testutils.LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "jsonnet_config",
			path:        "testdata/jsonnet.yaml",
			expectedDef: testutils.LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "log_level_off",
			path:        "testdata/logging_off.yaml",
			expectedDef: testutils.LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "tfstate_read",
			path:        "testdata/tfstate.yaml",
			expectedDef: testutils.LoadString(t, "testdata/tfstate.asl.json"),
		},
		{
			casename:    "yaml",
			path:        "testdata/yaml_def.yaml",
			expectedDef: testutils.LoadString(t, "testdata/hello_world.asl.json"),
			isYaml:      true,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			def, err := cfg.LoadDefinition()
			require.NoError(t, err)
			if c.isYaml {
				def = testutils.Yaml2Json(t, def)
			}
			require.JSONEq(t, c.expectedDef, def)
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
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.Error(t, err)
			if c.expected != "" {
				require.EqualError(t, err, c.expected)
			}
		})
	}

}
