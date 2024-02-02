package testutil

import (
	"bytes"
	"os"
	"testing"

	gc "github.com/kayac/go-config"
	"github.com/mashiike/stefunny/internal/jsonutil"
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

func YAML2JSON(t *testing.T, str string) string {
	t.Helper()
	j, err := jsonutil.YAML2JSON([]byte(str))
	if err != nil {
		t.Fatal(err)
	}
	return string(j)
}

func LoggerSetup(t *testing.T, minLevel string) func() {
	var buf bytes.Buffer
	logger.Setup(&buf, minLevel)
	return func() {
		logger.Setup(os.Stderr, minLevel)
		t.Log(buf.String())
	}
}
