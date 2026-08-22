package main

import (
	"path/filepath"
	"testing"
)

func TestRun(t *testing.T) {
	t.Setenv("AWS_REGION", "us-east-1")
	t.Setenv("AWS_ACCESS_KEY_ID", "dummy")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "dummy")
	cases := []struct {
		name string
		args []string
		code int
	}{
		{
			name: "render config",
			args: []string{"--log-level", "error", "--config", filepath.Join("..", "..", "testdata", "stefunny.yaml"), "render", "config"},
			code: 0,
		},
		{
			name: "config not found",
			args: []string{"--log-level", "error", "--config", filepath.Join("..", "..", "testdata", "not_found_dir", "stefunny.yaml"), "render", "config"},
			code: 1,
		},
		{
			name: "invalid ext-str",
			args: []string{"--log-level", "error", "--config", filepath.Join("..", "..", "testdata", "stefunny.yaml"), "--ext-str", "invalid", "render", "config"},
			code: 1,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			code := run(c.args)
			if code != c.code {
				t.Errorf("unexpected exit code: got %d, want %d", code, c.code)
			}
		})
	}
}
