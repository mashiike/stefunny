package stefunny_test

import (
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestAppRender(t *testing.T) {
	t.Setenv("START_AT", "Hello")
	t.Setenv("AWS_REGION", "us-east-1")
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/render"),
		goldie.WithNameSuffix(".golden.txt"),
	)
	cases := []struct {
		casename string
		path     string
		target   []string
		format   string
	}{
		{
			casename: "default",
			path:     "testdata/stefunny.yaml",
			target:   []string{"config", "definition"},
			format:   "",
		},
		{
			casename: "jsonnet",
			path:     "testdata/stefunny.yaml",
			target:   []string{"config", "definition"},
			format:   "jsonnet",
		},
		{
			casename: "yaml",
			path:     "testdata/stefunny.yaml",
			target:   []string{"config", "definition"},
			format:   "yaml",
		},
		{
			casename: "jsonnet_to_json",
			path:     "testdata/jsonnet_def.yaml",
			target:   []string{"definition"},
			format:   "json",
		},
		{
			casename: "full_def_to_jsonnet",
			path:     "testdata/full_def.yaml",
			target:   []string{"def", "config"},
			format:   "jsonnet",
		},
		{
			casename: "env_config",
			path:     "testdata/env_def.yaml",
			target:   []string{"def", "config"},
			format:   "jsonnet",
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			loc := dataloc.L(c.casename)
			t.Log("case location:", loc)
			LoggerSetup(t, "debug")
			m := NewMocks(t)
			defer m.AssertExpectations(t)
			app := newMockApp(t, c.path, m)
			ctx := context.Background()
			var buf bytes.Buffer
			err := app.Render(ctx, stefunny.RenderOption{
				Writer:  &buf,
				Targets: c.target,
				Format:  c.format,
			})
			require.NoError(t, err)
			g.Assert(t, c.casename, buf.Bytes())
		})
	}

}

func TestRendererTemplateize(t *testing.T) {
	t.Setenv("START_AT", "Hello")
	t.Setenv("AWS_REGION", "us-east-1")
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/render_templateize"),
		goldie.WithNameSuffix(".golden.txt"),
	)
	cases := []struct {
		casename string
		path     string
		format   string
		extStr   map[string]string
		extCode  map[string]string
	}{
		{
			casename: "env_config",
			path:     "testdata/env_def.yaml",
			format:   "jsonnet",
		},
		{
			casename: "tfstate",
			path:     "testdata/tfstate.yaml",
			format:   "jsonnet",
			extStr: map[string]string{
				"Comment": "great!!!",
			},
			extCode: map[string]string{
				"WaitSeconds": "60*2",
			},
		},
		{
			casename: "file_func",
			path:     "testdata/file_func.yaml",
			format:   "jsonnet",
		},
	}
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			loc := dataloc.L(c.casename)
			t.Log("case location:", loc)
			LoggerSetup(t, "debug")
			l := stefunny.NewConfigLoader(c.extStr, c.extCode)
			ctx := context.Background()
			cfg, err := l.Load(ctx, c.path)
			require.NoError(t, err)
			r := stefunny.NewRenderer(cfg)
			var buf bytes.Buffer
			fmt.Fprintln(&buf, "## config")
			err = r.RenderConfig(ctx, &buf, c.format, true)
			require.NoError(t, err)
			fmt.Fprintln(&buf, "## definition")
			err = r.RenderStateMachine(ctx, &buf, c.format, true)
			require.NoError(t, err)
			g.Assert(t, c.casename, buf.Bytes())
		})
	}
}
