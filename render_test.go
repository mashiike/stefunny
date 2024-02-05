package stefunny_test

import (
	"bytes"
	"context"
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
			l := stefunny.NewConfigLoader(nil, nil)
			ctx := context.Background()
			cfg, err := l.Load(ctx, c.path)
			require.NoError(t, err)
			mocks := NewMocks(t)
			defer mocks.AssertExpectations(t)
			app, err := stefunny.New(
				ctx, cfg,
				stefunny.WithEventBridgeClient(mocks.eventBridge),
				stefunny.WithSFnClient(mocks.sfn),
			)
			require.NoError(t, err)
			var buf bytes.Buffer
			err = app.Render(ctx, stefunny.RenderOption{
				Writer:  &buf,
				Targets: c.target,
				Format:  c.format,
			})
			require.NoError(t, err)
			g.Assert(t, c.casename, buf.Bytes())
		})
	}

}
