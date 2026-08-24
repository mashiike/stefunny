package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

type testBailout struct{}

func TestRun(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_SESSION_TOKEN", "")
	// A profile from the developer's environment would still be read by the SDK
	// shared config loader and can fail the run for reasons unrelated to the test.
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")

	configPath := filepath.Join("..", "..", "testdata", "stefunny.yaml")
	cases := []struct {
		name string
		// setup runs inside the subtest, after the shared env vars above are
		// applied, so it can override them (e.g. to simulate an environment
		// without usable AWS credentials).
		setup func(t *testing.T)
		args  []string
		// wantExitCode is the code run() must return. Only checked when
		// wantKongExit is false: when kong exits, the exit callback panics
		// before run() can return anything.
		wantExitCode int
		// wantKongExit is true if kong's own exit function must be called.
		// Parse errors never come back as a returned error in production, so
		// a case that expects one has to be pinned down here.
		wantKongExit bool
		wantOutput   []string
	}{
		{
			name:         "render config",
			args:         []string{"--log-level", "error", "--config", configPath, "render", "config"},
			wantExitCode: 0,
			wantOutput:   []string{"state_machine:", "name: Hello"},
		},
		{
			name: "render config without AWS credentials",
			setup: func(t *testing.T) {
				t.Setenv("AWS_ACCESS_KEY_ID", "")
				t.Setenv("AWS_SECRET_ACCESS_KEY", "")
				t.Setenv("AWS_SESSION_TOKEN", "")
				t.Setenv("AWS_PROFILE", "stefunny-test-profile-does-not-exist")
			},
			args:         []string{"--log-level", "error", "--config", configPath, "render", "config"},
			wantExitCode: 0,
			wantOutput:   []string{"state_machine:", "name: Hello"},
		},
		{
			name:         "config not found",
			args:         []string{"--log-level", "error", "--config", filepath.Join("..", "..", "testdata", "not_found_dir", "stefunny.yaml"), "render", "config"},
			wantExitCode: 1,
		},
		{
			name:         "invalid ext-str",
			args:         []string{"--log-level", "error", "--config", configPath, "--ext-str", "invalid", "render", "config"},
			wantExitCode: 1,
		},
		{
			name:         "unknown flag",
			args:         []string{"--no-such-flag"},
			wantKongExit: true,
			wantOutput:   []string{"unknown flag --no-such-flag"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.setup != nil {
				c.setup(t)
			}
			cli := stefunny.NewCLI()
			var buf bytes.Buffer
			cli.Writers(&buf, &buf)
			kongExitCalls := []int{}
			cli.Exit(func(code int) {
				kongExitCalls = append(kongExitCalls, code)
				panic(testBailout{})
			})

			var exitCode int
			func() {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(testBailout); !ok {
							require.FailNow(t, "unexpected panic", "%v", r)
						}
					}
				}()
				exitCode = run(cli, c.args)
			}()

			if c.wantKongExit {
				if len(kongExitCalls) != 1 {
					t.Errorf("kong exit calls: got %v, want exactly one call", kongExitCalls)
				} else if kongExitCalls[0] != 1 {
					t.Errorf("unexpected kong exit code: got %d, want 1", kongExitCalls[0])
				}
			} else if len(kongExitCalls) != 0 {
				t.Errorf("kong exited with %v, want no exit", kongExitCalls)
			}
			if !c.wantKongExit && exitCode != c.wantExitCode {
				t.Errorf("unexpected exit code: got %d, want %d", exitCode, c.wantExitCode)
			}
			out := buf.String()
			for _, want := range c.wantOutput {
				if !strings.Contains(out, want) {
					t.Errorf("output does not contain %q: %s", want, out)
				}
			}
		})
	}
}
