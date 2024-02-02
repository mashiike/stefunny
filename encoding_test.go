package stefunny_test

import (
	"testing"

	"github.com/mashiike/stefunny"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestYAML2JSON(t *testing.T) {
	yamlASL := LoadString(t, "testdata/hello_world.asl.yaml")
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := stefunny.YAML2JSON([]byte(yamlASL))
	require.NoError(t, err)
	require.JSONEq(t, jsonASL, string(bs))
}

func TestJSON2YAML(t *testing.T) {
	yamlASL := LoadString(t, "testdata/hello_world.asl.yaml")
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := stefunny.JSON2YAML([]byte(jsonASL))
	require.NoError(t, err)
	require.YAMLEq(t, yamlASL, string(bs))
}

func TestJSON2Jsonnet(t *testing.T) {
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := stefunny.JSON2Jsonnet("hello_world.asl.json", []byte(jsonASL))
	require.NoError(t, err)
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata"),
		goldie.WithNameSuffix(".golden.asl.jsonnet"),
	)
	g.Assert(t, "json2jsonnet", bs)
}
