package stefunny_test

import (
	"encoding/json"
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
