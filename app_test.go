package stefunny_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestAppRender(t *testing.T) {
	os.Setenv("START_AT", "Hello")
	cases := []struct {
		casename string
		path     string
		format   string
		expected string
	}{
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			expected: testutil.LoadString(t, "testdata/hello_world.dot"),
		},
		{
			casename: "jsonnet_config",
			path:     "testdata/jsonnet.yaml",
			expected: testutil.LoadString(t, "testdata/hello_world.dot"),
		},
		{
			casename: "full_def",
			path:     "testdata/full_def.yaml",
			expected: testutil.LoadString(t, "testdata/workflow1.dot"),
		},
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			format:   "json",
			expected: testutil.LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			format:   "yaml",
			expected: testutil.LoadString(t, "testdata/hello_world.asl.yaml"),
		},
		{
			casename: "env_config",
			path:     "testdata/env_def.yaml",
			format:   "json",
			expected: testutil.LoadString(t, "testdata/hello_world.asl.json"),
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path, stefunny.LoadConfigOption{
				TFState: "testdata/terraform.tfstate",
			})
			require.NoError(t, err)
			ctx := context.Background()
			app, err := stefunny.New(ctx, cfg)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = app.Render(ctx, stefunny.RenderOption{
				Writer: &buf,
				Format: c.format,
			})
			require.NoError(t, err)
			switch c.format {
			case "", "dot":
				require.ElementsMatch(t, strings.Split(c.expected, "\n"), strings.Split(buf.String(), "\n"))
			case "json":
				require.JSONEq(t, c.expected, buf.String())
			case "yaml":
				require.YAMLEq(t, c.expected, buf.String())
			}
		})
	}

}
