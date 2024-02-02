package stefunny_test

import (
	"bytes"
	"os"
	"testing"

	gc "github.com/kayac/go-config"
	"github.com/mashiike/stefunny/internal/logger"
)

func LoadString(t *testing.T, path string) string {
	t.Helper()
	bs, err := gc.ReadWithEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}

func LoggerSetup(t *testing.T, minLevel string) func() {
	var buf bytes.Buffer
	logger.Setup(&buf, minLevel)
	return func() {
		logger.Setup(os.Stderr, minLevel)
		t.Log(buf.String())
	}
}
