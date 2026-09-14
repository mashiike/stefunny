package stefunny_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/goccy/go-yaml"
	"github.com/mashiike/stefunny"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func TestYAMLToJSON(t *testing.T) {
	LoggerSetup(t, "debug")
	yamlASL := LoadString(t, "testdata/hello_world.asl.yaml")
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := yaml.YAMLToJSON([]byte(yamlASL))
	require.NoError(t, err)
	require.JSONEq(t, jsonASL, string(bs))
}

func TestJSONToYAML(t *testing.T) {
	yamlASL := LoadString(t, "testdata/hello_world.asl.yaml")
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := yaml.JSONToYAML([]byte(jsonASL))
	require.NoError(t, err)
	require.YAMLEq(t, yamlASL, string(bs))
}

func TestJSON2Jsonnet(t *testing.T) {
	jsonASL := LoadString(t, "testdata/hello_world.asl.json")
	bs, err := stefunny.JSON2Jsonnet("hello_world.asl.json", []byte(jsonASL))
	require.NoError(t, err)
	g := goldie.New(
		t,
		goldie.WithFixtureDir("testdata/encoding"),
		goldie.WithNameSuffix(".golden.asl.jsonnet"),
	)
	g.Assert(t, "json2jsonnet", bs)
}

func TestKeysToSnakeCase__CreateStateMachineInput(t *testing.T) {
	LoggerSetup(t, "debug")
	yamlStr := `
name: "test"
definition: "test.asl.json"
role_arn: "arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"
logging_configuration:
  include_execution_data: true
  level: "FATAL"
  destinations:
    - cloudwatch_logs_log_group:
        log_group_arn: "arn:aws:logs:ap-northeast-1:123456789012:log-group:test:*"
`
	var obj stefunny.KeysToSnakeCase[sfn.CreateStateMachineInput]
	err := yaml.UnmarshalWithOptions([]byte(yamlStr), &obj, yaml.UseJSONUnmarshaler())
	require.NoError(t, err)
	expected := sfn.CreateStateMachineInput{
		Name:       aws.String("test"),
		Definition: aws.String("test.asl.json"),
		RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
		LoggingConfiguration: &sfntypes.LoggingConfiguration{
			IncludeExecutionData: true,
			Level:                sfntypes.LogLevelFatal,
			Destinations: []sfntypes.LogDestination{
				{
					CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{
						LogGroupArn: aws.String("arn:aws:logs:ap-northeast-1:123456789012:log-group:test:*"),
					},
				},
			},
		},
	}
	require.EqualExportedValues(
		t,
		expected,
		obj.Value,
	)
	actualYAML, err := yaml.MarshalWithOptions(stefunny.NewKeysToSnakeCase(expected), yaml.UseJSONMarshaler())
	require.NoError(t, err)
	t.Log(string(actualYAML))
	require.YAMLEq(t, yamlStr, string(actualYAML))

	jsonBs, err := yaml.YAMLToJSON([]byte(yamlStr))
	require.NoError(t, err)
	t.Log(string(jsonBs))
	var obj2 stefunny.KeysToSnakeCase[sfn.CreateStateMachineInput]
	err = json.Unmarshal(jsonBs, &obj2)
	require.NoError(t, err)
	require.EqualExportedValues(
		t,
		expected,
		obj2.Value,
	)

	actualJSON, err := json.Marshal(stefunny.NewKeysToSnakeCase(expected))
	require.NoError(t, err)
	require.JSONEq(t, string(jsonBs), string(actualJSON))
}

func TestJSONDiffString__Ignore(t *testing.T) {
	from := `{"Name":"foo","Tags":{"Foo":"bar","Env":"dev"}}`
	to := `{"Name":"foo","Tags":{"Foo":"baz","Env":"prod"}}`

	t.Run("差分が消える", func(t *testing.T) {
		ds, err := stefunny.JSONDiffString(from, to, stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".Tags.Foo"))
		require.NoError(t, err)
		require.NotContains(t, ds, `"Foo"`)
		require.Contains(t, ds, `"Env"`)
	})

	t.Run("存在しないパスをignoreしてもno-opでエラーにならない", func(t *testing.T) {
		ds, err := stefunny.JSONDiffString(from, to, stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".NoSuchKey.Deep.Path"))
		require.NoError(t, err)
		require.Contains(t, ds, `"Foo"`)
		require.Contains(t, ds, `"Env"`)
	})

	t.Run("jqクエリの構文エラーはエラーを返す", func(t *testing.T) {
		_, err := stefunny.JSONDiffString(from, to, stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".foo["))
		require.Error(t, err)
	})

	t.Run("構文は正しいがdelのパスとして無効なクエリは実行時エラーを返す", func(t *testing.T) {
		_, err := stefunny.JSONDiffString(from, to, stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".Tags | keys"))
		require.Error(t, err)
	})

	t.Run("片側にしか存在しないキーをignoreすると差分が消える", func(t *testing.T) {
		from := `{"Name":"foo","Tags":{"Foo":"bar"}}`
		to := `{"Name":"foo","Tags":{"Foo":"bar","Env":"prod"}}`
		ds, err := stefunny.JSONDiffString(from, to, stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".Tags.Env"))
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(ds))
	})

	t.Run("null入力にignoreを指定してもエラーにならない", func(t *testing.T) {
		ds, err := stefunny.JSONDiffString("", "", stefunny.JSONDiffUnified(false), stefunny.JSONDiffIgnore(".Tags.Foo"))
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(ds))
	})
}
