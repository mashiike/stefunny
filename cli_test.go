package stefunny_test

import (
	"bytes"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/motemen/go-testutil/dataloc"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

type testBailout struct{}

func TestCLI__Parse(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	cases := []struct {
		name string
		args []string
		cmd  string
		code int
		envs map[string]string
	}{
		{
			name: "no args",
			args: []string{},
			code: 1,
		},
		{
			name: "unknown command",
			args: []string{"unknown"},
			code: 1,
		},
		{
			name: "help",
			args: []string{"--help"},
			code: 0,
		},
		{
			name: "version",
			args: []string{"--log-level", "warn", "version"},
			cmd:  "version",
		},
		{
			name: "version help",
			args: []string{"version", "--help"},
			code: 0,
		},
		{
			name: "init",
			args: []string{"init", "--state-machine", "test"},
			cmd:  "init",
		},
		{
			name: "init help",
			args: []string{"init", "--help"},
			code: 0,
		},
		{
			name: "init with config",
			args: []string{"init", "--state-machine", "test", "--config", "config.yaml"},
			cmd:  "init",
		},
		{
			name: "delete with log-level",
			args: []string{"delete", "--log-level", "debug"},
			cmd:  "delete",
		},
		{
			name: "delete with dry-run",
			args: []string{"delete", "--dry-run"},
			cmd:  "delete",
		},
		{
			name: "delete help",
			args: []string{"delete", "--help"},
			code: 0,
		},
		{
			name: "deploy dry run",
			args: []string{"deploy", "--dry-run"},
			cmd:  "deploy",
			envs: map[string]string{
				"AWS_REGION": "ap-northeast-1",
			},
		},
		{
			name: "deploy with region",
			args: []string{"deploy", "--region", "ap-northeast-1"},
			cmd:  "deploy",
		},
		{
			name: "deploy help",
			args: []string{"deploy", "--help"},
			code: 0,
		},
		{
			name: "schedule dry run",
			args: []string{"schedule", "--dry-run", "--enabled"},
			cmd:  "schedule",
		},
		{
			name: "schedule enable",
			args: []string{"schedule", "--enabled"},
			cmd:  "schedule",
		},
		{
			name: "schedule disable",
			args: []string{"schedule", "--disabled"},
			cmd:  "schedule",
		},
		{
			name: "schedule invalid",
			args: []string{"schedule", "--enabled", "--disabled"},
			code: 1,
		},
		{
			name: "schedule no flag",
			args: []string{"schedule"},
			code: 1,
		},
		{
			name: "schedule help",
			args: []string{"schedule", "--help"},
			code: 0,
		},
		{
			name: "render",
			args: []string{"render", "--log-level", "debug"},
			cmd:  "render",
		},
		{
			name: "render help",
			args: []string{"render", "--help"},
			code: 0,
		},
		{
			name: "render with format",
			args: []string{"render", "--format", "yaml"},
			cmd:  "render",
		},
		{
			name: "render invalid format",
			args: []string{"render", "--format", "invalid"},
			code: 1,
		},
		{
			name: "execute",
			args: []string{"execute", "--log-level", "debug"},
			cmd:  "execute",
		},
		{
			name: "execute help",
			args: []string{"execute", "--help"},
			code: 0,
		},
		{
			name: "execute with input",
			args: []string{"execute", "--input", "testdata/input.json"},
			cmd:  "execute",
		},
		{
			name: "execute with stdin",
			args: []string{"execute", "--input", "-"},
			cmd:  "execute",
		},
	}
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/cli"),
		goldie.WithSubTestNameForDir(true),
	)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			loc := dataloc.L(c.name)
			for k, v := range c.envs {
				t.Setenv(k, v)
			}
			cli := stefunny.NewCLI()
			var buf bytes.Buffer
			cli.Writers(&buf, &buf)
			cli.NoExpandPath()
			cli.Exit(func(code int) {
				require.Equal(t, c.code, code, "expected exit code: %s", loc)
				panic(testBailout{})
			})
			defer func() {
				if r := recover(); r != nil {
					if _, ok := r.(testBailout); !ok {
						require.FailNow(t, "unexpected panic: %v: %s", r, loc)
					}
				}
				t.Log("test case location:", loc)
				if buf.Len() > 0 {
					g.WithNameSuffix(".golden.txt")
					output := buf.Bytes()
					output = bytes.ReplaceAll(output, []byte(stefunny.Version), []byte("v*.*.*"))
					g.Assert(t, "console", output)
				}
				g.WithNameSuffix(".golden.json")
				g.AssertJson(t, "data", cli)
			}()
			cmd, err := cli.Parse(c.args)
			require.NoError(t, err)
			require.Equal(t, c.cmd, cmd, "expected command: %s", loc)
		})
	}
}
