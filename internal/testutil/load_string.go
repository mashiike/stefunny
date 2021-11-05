package testutil

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	gc "github.com/kayac/go-config"
	"github.com/mashiike/stefunny/internal/logger"
	"gopkg.in/yaml.v3"
)

func LoadString(t *testing.T, path string) string {
	t.Helper()
	bs, err := gc.ReadWithEnv(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bs)
}

func Yaml2Json(t *testing.T, str string) string {
	t.Helper()
	var temp map[string]interface{}
	if err := yaml.Unmarshal([]byte(str), &temp); err != nil {
		t.Fatal(err)
	}
	bs, err := json.Marshal(temp)
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
