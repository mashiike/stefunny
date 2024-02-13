package stefunny_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/mashiike/stefunny"
)

func LoadString(t *testing.T, path string) string {
	t.Helper()
	bs, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}

func LoggerSetup(t *testing.T, minLevel string) {
	t.Helper()
	var buf bytes.Buffer
	cleanup := stefunny.LoggerSetup(&buf, minLevel)
	t.Cleanup(cleanup)
}
