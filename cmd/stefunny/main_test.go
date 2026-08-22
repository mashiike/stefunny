package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mashiike/stefunny"
)

// stefunny.New eagerly resolves the AWS config and calls sts:GetCallerIdentity,
// so without a stub endpoint every case here would reach the real AWS API.
func stubSTS(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/xml")
		_, _ = w.Write([]byte(`<GetCallerIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <GetCallerIdentityResult>
    <Arn>arn:aws:iam::012345678901:user/dummy</Arn>
    <UserId>AIDADUMMY</UserId>
    <Account>012345678901</Account>
  </GetCallerIdentityResult>
</GetCallerIdentityResponse>`))
	}))
	t.Cleanup(server.Close)
	t.Setenv("AWS_ENDPOINT_URL_STS", server.URL)
}

func TestRun(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	t.Setenv("AWS_SESSION_TOKEN", "")
	// A profile from the developer's environment would still be read by the SDK
	// shared config loader and can fail the run for reasons unrelated to the test.
	t.Setenv("AWS_PROFILE", "")
	t.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	stubSTS(t)

	configPath := filepath.Join("..", "..", "testdata", "stefunny.yaml")
	kongExit := func(code int) *int { return &code }
	cases := []struct {
		name string
		args []string
		// wantExitCode is the code run() must return.
		wantExitCode int
		// wantKongExitCode is the code kong's own exit function must be called
		// with. Parse errors never come back as a returned error in production,
		// so a case that expects one has to pin the code down here; nil means the
		// arguments must parse and kong must not exit at all.
		wantKongExitCode *int
		wantOutput       []string
	}{
		{
			name:         "render config",
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
			name:             "unknown flag",
			args:             []string{"--no-such-flag"},
			wantExitCode:     1,
			wantKongExitCode: kongExit(1),
			wantOutput:       []string{"unknown flag --no-such-flag"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli := stefunny.NewCLI()
			var buf bytes.Buffer
			cli.Writers(&buf, &buf)
			kongExitCalls := []int{}
			cli.Exit(func(code int) {
				kongExitCalls = append(kongExitCalls, code)
			})

			exitCode := run(cli, c.args)
			if exitCode != c.wantExitCode {
				t.Errorf("unexpected exit code: got %d, want %d", exitCode, c.wantExitCode)
			}
			switch {
			case c.wantKongExitCode == nil && len(kongExitCalls) != 0:
				t.Errorf("kong exited with %v, want no exit", kongExitCalls)
			case c.wantKongExitCode != nil && len(kongExitCalls) != 1:
				t.Errorf("kong exit calls: got %v, want exactly one call with %d", kongExitCalls, *c.wantKongExitCode)
			case c.wantKongExitCode != nil && kongExitCalls[0] != *c.wantKongExitCode:
				t.Errorf("unexpected kong exit code: got %d, want %d", kongExitCalls[0], *c.wantKongExitCode)
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
