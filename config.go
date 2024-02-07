package stefunny

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fujiwara/tfstate-lookup/tfstate"
	jsonnet "github.com/google/go-jsonnet"
	gv "github.com/hashicorp/go-version"
	gc "github.com/kayac/go-config"
	"github.com/serenize/snaker"
	"gopkg.in/yaml.v3"
)

const (
	jsonnetExt = ".jsonnet"
	jsonExt    = ".json"
	ymlExt     = ".yml"
	yamlExt    = ".yaml"
)

type CloudWatchLogsClient interface {
	cloudwatchlogs.DescribeLogGroupsAPIClient
}
type ConfigLoader struct {
	loader       *gc.Loader
	funcMap      template.FuncMap
	vm           *jsonnet.VM
	cwLogsClient CloudWatchLogsClient
}

func NewConfigLoader(extStr, extCode map[string]string) *ConfigLoader {
	vm := jsonnet.MakeVM()
	for k, v := range extStr {
		vm.ExtVar(k, v)
	}
	for k, v := range extCode {
		vm.ExtCode(k, v)
	}
	return &ConfigLoader{
		loader:  gc.New(),
		funcMap: make(template.FuncMap),
		vm:      vm,
	}
}

func (l *ConfigLoader) SetCloudWatchLogsClient(client CloudWatchLogsClient) {
	l.cwLogsClient = client
}

func (l *ConfigLoader) AppendTFState(ctx context.Context, prefix string, tfState string) error {
	funcs, err := tfstate.FuncMap(ctx, tfState)
	if err != nil {
		return fmt.Errorf("tfstate %w", err)
	}
	return l.AppendFuncMap(prefix, funcs)
}

func (l *ConfigLoader) AppendFuncMap(prefix string, funcMap template.FuncMap) error {
	appendTarget := make(template.FuncMap, len(funcMap))
	for k, v := range funcMap {
		modifiedKey := prefix + k
		if _, ok := l.funcMap[modifiedKey]; ok {
			return fmt.Errorf("funcMap key %s already exists", modifiedKey)
		}
		l.funcMap[modifiedKey] = v
		appendTarget[modifiedKey] = v
	}
	l.loader.Funcs(appendTarget)
	return nil
}

func (l *ConfigLoader) load(path string, strict bool, withEnv bool, v any) error {
	ext := filepath.Ext(path)
	switch ext {
	case yamlExt, ymlExt:
		var b []byte
		var err error
		if withEnv {
			b, err = l.loader.ReadWithEnv(path)
		} else {
			b, err = os.ReadFile(path)
		}
		if err != nil {
			return err
		}

		dec := yaml.NewDecoder(bytes.NewReader(b))
		if strict {
			dec.KnownFields(true)
		}
		if err := dec.Decode(v); err != nil {
			return err
		}
		return nil
	case jsonExt, jsonnetExt:
		jsonStr, err := l.vm.EvaluateFile(path)
		if err != nil {
			return fmt.Errorf("failed to evaluate jsonnet file: %w", err)
		}
		b := []byte(jsonStr)
		if withEnv {
			b, err = l.loader.ReadWithEnvBytes([]byte(jsonStr))
			if err != nil {
				return fmt.Errorf("failed to read template file: %w", err)
			}
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		if strict {
			dec.DisallowUnknownFields()
		}
		if err := dec.Decode(v); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func (l *ConfigLoader) Load(ctx context.Context, path string) (*Config, error) {
	cfg := NewDefaultConfig()
	cfg.ConfigDir = filepath.Dir(path)
	// pre load for tfstate path read
	if err := l.load(path, false, false, cfg); err != nil {
		return nil, fmt.Errorf("pre load config `%s`: %w", path, err)
	}
	for i, tfstate := range cfg.TFState {
		var loc string
		if tfstate.URL != "" {
			u, err := url.Parse(tfstate.URL)
			if err != nil {
				return nil, fmt.Errorf("tfstate[%d].url parse error: %w", i, err)
			}
			if u.Scheme == "" {
				tfstate.Path = tfstate.URL
			}
		}
		if tfstate.Path != "" {
			loc = tfstate.Path
			if !filepath.IsAbs(loc) {
				loc = filepath.Join(filepath.Dir(path), loc)
			}
		}
		if loc == "" {
			return nil, fmt.Errorf("tfstate[%d].path or tfstate[%d].url is required", i, i)
		}
		if err := l.AppendTFState(ctx, tfstate.FuncPrefix, loc); err != nil {
			return nil, fmt.Errorf("tfstate[%d] %w", i, err)
		}
	}

	cfg.StateMachine.Strict = true
	if err := l.load(path, true, true, cfg); err != nil {
		return nil, fmt.Errorf("load config `%s`: %w", path, err)
	}
	// migration from old version: TODO delete v0.7.0

	if cfg.StateMachine != nil && cfg.StateMachine.Logging != nil {
		log.Println("[warn] state_machine.logging is deprecated. Use state_machine.logging_configuration instead. (since v0.6.0)")
		cfg.StateMachine.Value.LoggingConfiguration = &sfntypes.LoggingConfiguration{
			Level:                sfntypes.LogLevel(cfg.StateMachine.Logging.Level),
			IncludeExecutionData: cfg.StateMachine.Logging.IncludeExecutionData,
		}
		if cfg.StateMachine.Logging.Destination != nil {
			if _, err := arn.Parse(cfg.StateMachine.Logging.Destination.LogGroup); err != nil {
				client := l.cwLogsClient
				if client == nil {
					awsCfg, err := cfg.LoadAWSConfig(ctx)
					if err != nil {
						return nil, fmt.Errorf("load aws config:%w", err)
					}
					client = cloudwatchlogs.NewFromConfig(awsCfg)
				}
				p := cloudwatchlogs.NewDescribeLogGroupsPaginator(client, &cloudwatchlogs.DescribeLogGroupsInput{
					Limit: aws.Int32(50),
				})
				fround := false
				for p.HasMorePages() {
					page, err := p.NextPage(ctx)
					if err != nil {
						return nil, fmt.Errorf("describe log groups:%w", err)
					}
					for _, g := range page.LogGroups {
						if *g.LogGroupName == cfg.StateMachine.Logging.Destination.LogGroup {
							cfg.StateMachine.Value.LoggingConfiguration.Destinations = append(cfg.StateMachine.Value.LoggingConfiguration.Destinations, sfntypes.LogDestination{
								CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{
									LogGroupArn: g.Arn,
								},
							})
							fround = true
							break
						}
					}
				}
				if !fround {
					return nil, fmt.Errorf("log group `%s` not found", cfg.StateMachine.Logging.Destination.LogGroup)
				}
			} else {
				cfg.StateMachine.Value.LoggingConfiguration.Destinations = append(cfg.StateMachine.Value.LoggingConfiguration.Destinations, sfntypes.LogDestination{
					CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{
						LogGroupArn: aws.String(cfg.StateMachine.Logging.Destination.LogGroup),
					},
				})
			}
		}
	}
	if cfg.StateMachine != nil && cfg.StateMachine.Tracing != nil {
		log.Println("[warn] state_machine.tracing is deprecated. Use state_machine.tracing_configuration instead. (since v0.6.0)")
		cfg.StateMachine.Value.TracingConfiguration = &sfntypes.TracingConfiguration{
			Enabled: cfg.StateMachine.Tracing.Enabled,
		}
	}
	if cfg.Schedule != nil {
		log.Println("[warn] schedule is deprecated. Use trigger.schedule or trigger.event instead. (since v0.6.0)")
		if cfg.Trigger == nil {
			cfg.Trigger = &TriggerConfig{}
		}
		for _, s := range cfg.Schedule {
			event := TriggerEventConfig{
				KeysToSnakeCase: NewKeysToSnakeCase(EventBridgeRule{
					PutRuleInput: eventbridge.PutRuleInput{
						Name:               &s.RuleName,
						ScheduleExpression: &s.Expression,
						Description:        &s.Description,
						RoleArn:            &s.RoleArn,
					},
				}),
			}
			cfg.Trigger.Event = append(cfg.Trigger.Event, event)
		}
		cfg.Schedule = nil
	}
	if err := cfg.Restrict(); err != nil {
		return nil, fmt.Errorf("config restrict:%w", err)
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		return nil, fmt.Errorf("config validate version:%w", err)
	}
	if cfg.StateMachine.Value.Definition != nil {
		return cfg, nil
	}
	if cfg.StateMachine.DefinitionPath == "" {
		return nil, errors.New("state_machine.definition is required")
	}
	// cfg.StateMachine.Definition written definition file path
	var definition JSONRawMessage
	definitionPath := filepath.Clean(filepath.Join(filepath.Dir(path), cfg.StateMachine.DefinitionPath))
	log.Println("[debug] definition path =", definitionPath)
	if err := l.load(definitionPath, false, true, &definition); err != nil {
		return nil, fmt.Errorf("load definition `%s`: %w", definitionPath, err)
	}
	cfg.StateMachine.Value.Definition = aws.String(string(definition))
	return cfg, nil
}

type Config struct {
	RequiredVersion string `yaml:"required_version,omitempty" json:"required_version,omitempty" toml:"required_version,omitempty" env:"REQUIRED_VERSION" validate:"omitempty,version"`
	AWSRegion       string `yaml:"aws_region,omitempty" json:"aws_region,omitempty" toml:"aws_region,omitempty" env:"AWS_REGION" validate:"omitempty,region"`

	StateMachine *StateMachineConfig `yaml:"state_machine,omitempty" json:"state_machine,omitempty"`
	Trigger      *TriggerConfig      `yaml:"trigger,omitempty" json:"trigger,omitempty"`
	Schedule     []*ScheduleConfig   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Tags         map[string]string   `yaml:"tags,omitempty" json:"tags,omitempty"`

	Endpoints *EndpointsConfig `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	TFState []*TFStateConfig `yaml:"tfstate,omitempty" json:"tfstate,omitempty"`

	ConfigDir string `yaml:"-"`
	//private field
	mu                 sync.Mutex
	versionConstraints gv.Constraints `yaml:"-,omitempty"`
	awsCfg             *aws.Config    `yaml:"-"`
}

type TFStateConfig struct {
	FuncPrefix string `yaml:"func_prefix,omitempty" json:"func_prefix,omitempty"`
	Path       string `yaml:"path,omitempty" json:"path,omitempty"`
	URL        string `yaml:"url,omitempty" json:"url,omitempty"`
}

type StateMachineConfig struct {
	KeysToSnakeCase[sfn.CreateStateMachineInput] `yaml:",inline" json:",inline"`
	DefinitionPath                               string `yaml:"definition_path,omitempty" json:"definition_path,omitempty"`

	Logging *StateMachineLogging           `yaml:"-,omitempty"`
	Tracing *sfntypes.TracingConfiguration `yaml:"-,omitempty"`
}

type StateMachineLogging struct {
	Level                string                          `yaml:"level,omitempty" json:"level,omitempty"`
	IncludeExecutionData bool                            `yaml:"include_execution_data,omitempty" json:"include_execution_data,omitempty"`
	Destination          *StateMachineLoggingDestination `yaml:"destination,omitempty" json:"destination,omitempty"`
}

type StateMachineLoggingDestination struct {
	LogGroup string `yaml:"log_group,omitempty" json:"log_group,omitempty"`
}

func (cfg *StateMachineConfig) UnmarshalYAML(node *yaml.Node) error {
	var data map[string]interface{}
	if err := node.Decode(&data); err != nil {
		return err
	}
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bs, cfg); err != nil {
		return err
	}
	return nil
}

func (cfg *StateMachineConfig) UnmarshalJSON(b []byte) error {
	var data map[string]json.RawMessage
	if err := json.Unmarshal(b, &data); err != nil {
		return err
	}
	if logging, ok := data["logging"]; ok {
		if err := json.Unmarshal(logging, &cfg.Logging); err != nil {
			return fmt.Errorf("logging unmarshal failed:%w", err)
		}
		delete(data, "logging")
	}
	if tracing, ok := data["tracing"]; ok {
		if err := json.Unmarshal(tracing, &cfg.Tracing); err != nil {
			return fmt.Errorf("tracing unmarshal failed:%w", err)
		}
		delete(data, "tracing")
	}
	replaced, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("replaced unmarshal failed:%w", err)
	}
	if err := json.Unmarshal(replaced, &cfg.KeysToSnakeCase); err != nil {
		return fmt.Errorf("replaced unmarshal failed:%w", err)
	}
	if cfg.Value.Definition != nil {
		if json.Valid([]byte(*cfg.Value.Definition)) {
			return nil
		}
		cfg.DefinitionPath = *cfg.Value.Definition
		cfg.Value.Definition = nil
	}
	return nil
}

type EndpointsConfig struct {
	StepFunctions  string `yaml:"stepfunctions,omitempty" json:"step_functions,omitempty"`
	CloudWatchLogs string `yaml:"cloudwatchlogs,omitempty" json:"cloud_watch_logs,omitempty"`
	STS            string `yaml:"sts,omitempty" json:"sts,omitempty"`
	EventBridge    string `yaml:"eventbridge,omitempty" json:"event_bridge,omitempty"`
}

type ScheduleConfig struct {
	ID          string `yaml:"id,omitempty" json:"id,omitempty"`
	RuleName    string `yaml:"rule_name,omitempty" json:"rule_name,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Expression  string `yaml:"expression,omitempty" json:"expression,omitempty"`
	RoleArn     string `yaml:"role_arn,omitempty" json:"role_arn,omitempty"`
}

type TriggerConfig struct {
	Schedule []TriggerScheduleConfig `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Event    []TriggerEventConfig    `yaml:"event,omitempty" json:"event,omitempty"`
}

type TriggerScheduleConfig struct {
}

type TriggerEventConfig struct {
	KeysToSnakeCase[EventBridgeRule] `yaml:",inline" json:",inline"`
}

// Restrict restricts a configuration.
func (cfg *Config) Restrict() error {
	if cfg.RequiredVersion != "" {
		constraints, err := gv.NewConstraint(cfg.RequiredVersion)
		if err != nil {
			return fmt.Errorf("required_version has invalid format: %w", err)
		}
		cfg.versionConstraints = constraints
	}
	if cfg.StateMachine == nil {
		return errors.New("state_machine is required")
	}
	if err := cfg.StateMachine.Restrict(); err != nil {
		return fmt.Errorf("state_machine.%w", err)
	}
	if cfg.Trigger != nil {
		if err := cfg.Trigger.Restrict(); err != nil {
			return fmt.Errorf("trigger.%w", err)
		}
	}

	if len(cfg.Schedule) != 0 {
		for i, s := range cfg.Schedule {
			if err := s.Restrict(i, *cfg.StateMachine.Value.Name); err != nil {
				return fmt.Errorf("schedule[%d].%w", i, err)
			}
		}
	}
	if len(cfg.Tags) > 0 {
		log.Println("[warn] tags is deprecated. Use state_machine.tags instead. (since v0.6.0)")
	}
	return nil
}

func (cfg *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if cfg.awsCfg != nil {
		return *cfg.awsCfg, nil
	}
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.AWSRegion),
	}
	if endpointsResolver, ok := cfg.EndpointResolver(); ok {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(endpointsResolver))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}
	cfg.awsCfg = &awsCfg
	return awsCfg, nil
}

func (cfg *Config) StateMachineName() string {
	return *cfg.StateMachine.Value.Name
}

func (cfg *Config) StateMachineDefinition() string {
	return *cfg.StateMachine.Value.Definition
}

func (cfg *Config) NewCreateStateMachineInput() sfn.CreateStateMachineInput {
	input := cfg.StateMachine.Value
	found := false
	for _, tag := range input.Tags {
		if tag.Key == nil {
			continue
		}
		if *tag.Key == tagManagedBy {
			tag.Value = aws.String(appName)
			found = true
		}
	}
	if !found {
		input.Tags = append(input.Tags, sfntypes.Tag{
			Key:   aws.String(tagManagedBy),
			Value: aws.String(appName),
		})
	}
	return input
}

func (cfg *TriggerConfig) Restrict() error {
	return nil
}

func (cfg *Config) NewEventBridgeRules() EventBridgeRules {
	if cfg.Trigger == nil {
		return EventBridgeRules{}
	}
	rules := make(EventBridgeRules, 0, len(cfg.Trigger.Event))
	for _, e := range cfg.Trigger.Event {
		rules = append(rules, ptr(e.Value))
	}
	return rules
}

func (cfg *StateMachineConfig) SetDetinitionPath(path string) {
	cfg.DefinitionPath = path
}

func (cfg *StateMachineConfig) SetDefinition(definition string) {
	cfg.Value.Definition = aws.String(definition)
}

// Restrict restricts a configuration.
func (cfg *StateMachineConfig) Restrict() error {
	if cfg.Value.Name == nil || *cfg.Value.Name == "" {
		return errors.New("name is required")
	}
	if cfg.Value.RoleArn == nil || *cfg.Value.RoleArn == "" {
		return errors.New("role_arn is required")
	}
	if cfg.Value.Type == "" {
		cfg.Value.Type = sfntypes.StateMachineTypeStandard
	} else {
		var err error
		cfg.Value.Type, err = restrictSFnStateMachineType(string(cfg.Value.Type))
		if err != nil {
			return fmt.Errorf("type is %w", err)
		}
	}
	if cfg.Value.LoggingConfiguration == nil {
		return errors.New("logging_configuration is required")
	}
	if cfg.Value.LoggingConfiguration.Level == "" {
		cfg.Value.LoggingConfiguration.Level = sfntypes.LogLevelOff
	} else {
		var err error
		cfg.Value.LoggingConfiguration.Level, err = restrictLogLevel(string(cfg.Value.LoggingConfiguration.Level))
		if err != nil {
			return fmt.Errorf("logging_configuration.level is %w", err)
		}
	}
	for i, dest := range cfg.Value.LoggingConfiguration.Destinations {
		if dest.CloudWatchLogsLogGroup == nil {
			return fmt.Errorf("logging_configuration.destinations[%d].cloudwatch_logs_log_group is required", i)
		}
		if dest.CloudWatchLogsLogGroup.LogGroupArn == nil || *dest.CloudWatchLogsLogGroup.LogGroupArn == "" {
			return fmt.Errorf("logging_configuration.destinations[%d].cloudwatch_logs_log_group.log_group_arn is required", i)
		}
		logGroupARN, err := arn.Parse(*dest.CloudWatchLogsLogGroup.LogGroupArn)
		if err != nil {
			return fmt.Errorf(
				"logging_configuration.destinations[%d].cloudwatch_logs_log_group.log_group_arn = `%s` is invalid: %w",
				i, *dest.CloudWatchLogsLogGroup.LogGroupArn, err,
			)
		}
		if logGroupARN.Service != "logs" {
			return fmt.Errorf("logging_configuration.destinations[%d].cloudwatch_logs_log_group.log_group_arn is not CloudWatch Logs ARN", i)
		}
	}
	if cfg.Value.TracingConfiguration == nil {
		cfg.Value.TracingConfiguration = &sfntypes.TracingConfiguration{
			Enabled: false,
		}
	}
	if cfg.Value.Publish {
		cfg.Value.Publish = false
	}
	if cfg.Value.VersionDescription != nil {
		cfg.Value.VersionDescription = nil
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *ScheduleConfig) Restrict(index int, stateMachineName string) error {
	if cfg.RuleName == "" {
		middle := snaker.CamelToSnake(stateMachineName)
		cfg.RuleName = fmt.Sprintf("%s-%s-schedule", appName, middle)
		if index != 0 {
			cfg.RuleName += fmt.Sprintf("%d", index)
		}
	}
	if cfg.Expression == "" {
		return errors.New("expression is required")
	}
	if cfg.RoleArn == "" {
		return errors.New("role_arn is required")
	}
	return nil
}

func restrictSFnStateMachineType(tstr string) (sfntypes.StateMachineType, error) {
	t := sfntypes.StateMachineType(strings.ToUpper(tstr))
	typeValues := t.Values()
	str := "invalid type: please "
	for i, v := range typeValues {
		if t == v {
			return v, nil
		}
		str += string(v)
		if i < len(typeValues)-1 {
			str += ", "
		}
		if len(typeValues) >= 3 && i == len(typeValues)-2 {
			str += "or "
		}
	}
	return "", errors.New(str)
}

func restrictLogLevel(lstr string) (sfntypes.LogLevel, error) {
	l := sfntypes.LogLevel(strings.ToUpper(lstr))
	levelValues := l.Values()
	str := "invalid level: please "
	for i, v := range levelValues {
		if l == v {
			return v, nil
		}
		str += string(v)
		if i < len(levelValues)-1 {
			str += ", "
		}
		if len(levelValues) >= 3 && i == len(levelValues)-2 {
			str += "or "
		}
	}
	return "", errors.New(str)
}

// ValidateVersion validates a version satisfies required_version.
func (cfg *Config) ValidateVersion(version string) error {
	if cfg.versionConstraints == nil {
		log.Println("[warn] required_version is empty. Skip checking required_version.")
		return nil
	}
	versionParts := strings.SplitN(version, "-", 2)
	v, err := gv.NewVersion(versionParts[0])
	if err != nil {
		log.Printf("[warn] Invalid version format \"%s\". Skip checking required_version.", version)
		// invalid version string (e.g. "current") always allowed
		return nil
	}
	if !cfg.versionConstraints.Check(v) {
		return fmt.Errorf("version %s does not satisfy constraints required_version: %s", version, cfg.versionConstraints)
	}
	return nil
}

func NewDefaultConfig() *Config {
	return &Config{
		AWSRegion: os.Getenv("AWS_REGION"),
		StateMachine: &StateMachineConfig{
			KeysToSnakeCase: NewKeysToSnakeCase(sfn.CreateStateMachineInput{
				Type: sfntypes.StateMachineTypeStandard,
				LoggingConfiguration: &sfntypes.LoggingConfiguration{
					Level: sfntypes.LogLevelOff,
				},
				TracingConfiguration: &sfntypes.TracingConfiguration{
					Enabled: false,
				},
			}),
		},
		Tags: make(map[string]string),
	}
}

func (cfg *Config) EndpointResolver() (aws.EndpointResolverWithOptions, bool) {
	if cfg.Endpoints == nil {
		return nil, false
	}
	return aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		if cfg.AWSRegion != region {
			return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		}
		switch service {
		case sfn.ServiceID:
			if cfg.Endpoints.StepFunctions != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.Endpoints.StepFunctions,
					SigningRegion: cfg.AWSRegion,
				}, nil
			}
		case cloudwatchlogs.ServiceID:
			if cfg.Endpoints.StepFunctions != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.Endpoints.CloudWatchLogs,
					SigningRegion: cfg.AWSRegion,
				}, nil
			}
		case sts.ServiceID:
			if cfg.Endpoints.StepFunctions != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.Endpoints.STS,
					SigningRegion: cfg.AWSRegion,
				}, nil
			}
		case eventbridge.ServiceID:
			if cfg.Endpoints.StepFunctions != "" {
				return aws.Endpoint{
					PartitionID:   "aws",
					URL:           cfg.Endpoints.EventBridge,
					SigningRegion: cfg.AWSRegion,
				}, nil
			}
		}
		return aws.Endpoint{}, &aws.EndpointNotFoundError{}

	}), true
}
