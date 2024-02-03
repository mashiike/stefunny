package stefunny_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
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
			path:     "testdata/stefunny.yaml",
			expected: LoadString(t, "testdata/hello_world.dot"),
			format:   "dot",
		},
		{
			casename: "jsonnet_config",
			path:     "testdata/jsonnet_def.yaml",
			expected: LoadString(t, "testdata/hello_world.dot"),
			format:   "dot",
		},
		{
			casename: "full_def",
			path:     "testdata/full_def.yaml",
			expected: LoadString(t, "testdata/workflow1.dot"),
			format:   "dot",
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			format:   "json",
			expected: LoadString(t, "testdata/hello_world.asl.json"),
		},
		{
			casename: "default_config",
			path:     "testdata/stefunny.yaml",
			format:   "yaml",
			expected: LoadString(t, "testdata/hello_world.asl.yaml"),
		},
		{
			casename: "env_config",
			path:     "testdata/env_def.yaml",
			expected: LoadString(t, "testdata/hello_world.asl.json"),
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			loc := dataloc.L(c.casename)
			t.Log("case location:", loc)
			LoggerSetup(t, "debug")
			l := stefunny.NewConfigLoader(nil, nil)
			ctx := context.Background()
			err := l.AppendTFState(ctx, "", "testdata/terraform.tfstate")
			require.NoError(t, err)
			cfg, err := l.Load(c.path)
			require.NoError(t, err)
			app, err := stefunny.New(ctx, cfg)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = app.Render(ctx, stefunny.RenderOption{
				Writer: &buf,
				Format: c.format,
			})
			require.NoError(t, err)
			switch c.format {
			case "dot":
				require.ElementsMatch(t, strings.Split(c.expected, "\n"), strings.Split(buf.String(), "\n"))
			case "", "json":
				require.JSONEq(t, c.expected, buf.String())
			case "yaml":
				require.YAMLEq(t, c.expected, buf.String())
			}
		})
	}

}
