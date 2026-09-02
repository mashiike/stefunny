package stefunny_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		{
			casename: "zero_value_json",
			path:     "testdata/zero_value_def.yaml",
			target:   []string{"definition"},
			format:   "json",
		},
		{
			casename: "zero_value_jsonnet",
			path:     "testdata/zero_value_def.yaml",
			target:   []string{"definition"},
			format:   "jsonnet",
		},
		{
			casename: "zero_value_yaml",
			path:     "testdata/zero_value_def.yaml",
			target:   []string{"definition"},
			format:   "yaml",
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			loc := dataloc.L(c.casename)
			t.Log("case location:", loc)
			LoggerSetup(t, "debug")
			m := NewMocks(t)
			defer m.Finish()
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
		{
			casename: "zero_value",
			path:     "testdata/zero_value_def.yaml",
			format:   "json",
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

func TestRendererTemplateizeTFStateAbsoluteFileURL(t *testing.T) {
	t.Setenv("START_AT", "Hello")
	t.Setenv("AWS_REGION", "us-east-1")
	LoggerSetup(t, "debug")

	testdataDir, err := filepath.Abs("testdata")
	require.NoError(t, err)
	definitionPath := filepath.Join(testdataDir, "hello_world.asl.json")
	tfstatePath := filepath.Join(testdataDir, "terraform.tfstate")
	configPath := filepath.Join(t.TempDir(), "stefunny.yaml")
	configContent := fmt.Sprintf(`required_version: ">v0.0.0"

state_machine:
  name: Hello
  definition: %s
  role_arn: "{{ tfstate %saws_iam_role.test.arn%s }}"

tfstate:
  - location: file://%s
`, definitionPath, "`", "`", tfstatePath)
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	l := stefunny.NewConfigLoader(nil, nil)
	ctx := context.Background()
	cfg, err := l.Load(ctx, configPath)
	require.NoError(t, err)
	require.Equal(t, "arn:aws:iam::000000000000:role/test", *cfg.StateMachine.Value.RoleArn)

	r := stefunny.NewRenderer(cfg)
	var buf bytes.Buffer
	err = r.RenderConfig(ctx, &buf, "json", true)
	require.NoError(t, err)
	require.Contains(t, buf.String(), "{{ tfstate `aws_iam_role.test.arn` }}")
}

func TestRenderConfig_StillPrunesZeroValues(t *testing.T) {
	LoggerSetup(t, "debug")

	definitionPath, err := filepath.Abs("testdata/hello_world.asl.json")
	require.NoError(t, err)
	configPath := filepath.Join(t.TempDir(), "stefunny.yaml")
	configContent := fmt.Sprintf(`required_version: ">v0.0.0"

state_machine:
  name: Hello
  definition: %s
  role_arn: arn:aws:iam::012345678901:role/service-role/StepFunctions-Hello-role

tags:
  empty_tag: ""
  real_tag: "value"
`, definitionPath)
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0600))

	l := stefunny.NewConfigLoader(nil, nil)
	ctx := context.Background()
	cfg, err := l.Load(ctx, configPath)
	require.NoError(t, err)

	r := stefunny.NewRenderer(cfg)
	var buf bytes.Buffer
	err = r.RenderConfig(ctx, &buf, "json", false)
	require.NoError(t, err)
	require.NotContains(t, buf.String(), "empty_tag")
	require.Contains(t, buf.String(), "real_tag")
}
