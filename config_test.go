package stefunny_test

import (
	"testing"

	gc "github.com/kayac/go-config"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestConfigLoadValid(t *testing.T) {
	cases := []struct {
		casename    string
		path        string
		expectedDef string
	}{
		{
			casename:    "default_config",
			path:        "testdata/default.yaml",
			expectedDef: loadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "jsonnet_config",
			path:        "testdata/jsonnet.yaml",
			expectedDef: loadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "log_level_off",
			path:        "testdata/logging_off.yaml",
			expectedDef: loadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename:    "tfstate_read",
			path:        "testdata/tfstate.yaml",
			expectedDef: loadString(t, "testdata/tfstate.asl.json"),
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			def, err := cfg.LoadDefinition()
			require.NoError(t, err)
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

func loadString(t *testing.T, path string) string {
	t.Helper()
	bs, err := gc.ReadWithEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}
