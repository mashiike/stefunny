package testutils

import (
	"encoding/json"
	"testing"

	gc "github.com/kayac/go-config"
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
