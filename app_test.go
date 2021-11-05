package stefunny_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/testutil"
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
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			testutil.LoggerSetup(t, "debug")
			cfg := stefunny.NewDefaultConfig()
			err := cfg.Load(c.path)
			require.NoError(t, err)
			ctx := context.Background()
			app, err := stefunny.New(ctx, cfg)
			require.NoError(t, err)
			var buf bytes.Buffer
			app.Render(ctx, stefunny.RenderOption{
				Writer: &buf,
			})
			require.ElementsMatch(t, strings.Split(c.expected, "\n"), strings.Split(buf.String(), "\n"))
		})
	}

}
