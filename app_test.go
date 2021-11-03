package sffle_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mashiike/sffle"
	"github.com/stretchr/testify/require"
)

func TestAppRender(t *testing.T) {
	cases := []struct {
		casename string
		path     string
		expected string
	}{
		{
			casename: "default_config",
			path:     "testdata/default.yaml",
			expected: loadString(t, "testdata/hello_world.dot"),
		},
		{
			casename: "jsonnet_config",
			path:     "testdata/jsonnet.yaml",
			expected: loadString(t, "testdata/hello_world.dot"),
		},
		{
			casename: "full_def",
			path:     "testdata/full_def.yaml",
			expected: loadString(t, "testdata/workflow1.dot"),
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			cfg := sffle.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			ctx := context.Background()
			app, err := sffle.New(ctx, cfg)
			require.NoError(t, err)
			var buf bytes.Buffer
			app.Render(ctx, sffle.RenderOption{
				Writer: &buf,
			})
			require.ElementsMatch(t, strings.Split(c.expected, "\n"), strings.Split(buf.String(), "\n"))
		})
	}

}
