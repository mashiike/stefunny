package stefunny_test

import (
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadValid(t *testing.T) {
	cases := []struct {
		casename    string
		path        string
		expectedDef string
		isYaml      bool
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
			isYaml:      true,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			LoggerSetup(t, "debug")
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path, stefunny.LoadConfigOption{
				TFState: "testdata/terraform.tfstate",
				ExtStr:  c.extStr,
				ExtCode: c.extCode,
			})
			require.NoError(t, err)
			def, err := cfg.LoadDefinition()
			require.NoError(t, err)
			if c.isYaml {
				bs, err := stefunny.YAML2JSON([]byte(def))
				require.NoError(t, err)
				def = string(bs)
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
			LoggerSetup(t, "debug")
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path, stefunny.LoadConfigOption{
				TFState: "testdata/terraform.tfstate",
			})
			require.Error(t, err)
			if c.expected != "" {
				require.EqualError(t, err, c.expected)
			}
		})
	}

}
