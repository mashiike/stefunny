package stefunny

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
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

type ConfigLoader struct {
	loader  *gc.Loader
	funcMap template.FuncMap
	vm      *jsonnet.VM
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
			b, err = os.ReadFile(path)
		} else {
			b, err = l.loader.ReadWithEnv(path)
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

type jsonRawMessage json.RawMessage

func (j *jsonRawMessage) UnmarshalYAML(value *yaml.Node) error {
	var data any
	if err := value.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode yaml node: %w", err)
	}
	m, err := convertKeyString(data)
	if err != nil {
		return fmt.Errorf("failed to convert key string: %w", err)
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal json: %w", err)
	}
	*j = jsonRawMessage(bs)
	return nil
}

func (j *jsonRawMessage) UnmarshalJSON(bs []byte) error {
	var raw json.RawMessage
	if err := json.Unmarshal(bs, &raw); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}
	*j = jsonRawMessage(raw)
	return nil
}

func (l *ConfigLoader) Load(path string) (*Config, error) {
	cfg := NewDefaultConfig()
	if err := l.load(path, true, true, cfg); err != nil {
		return nil, fmt.Errorf("load config `%s`: %w", path, err)
	}
	if err := cfg.Restrict(); err != nil {
		return nil, fmt.Errorf("config restrict:%w", err)
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		return nil, fmt.Errorf("config validate version:%w", err)
	}
	if cfg.StateMachine.Definition == "" {
		return nil, errors.New("state_machine.definition is required")
	}
	if json.Valid([]byte(cfg.StateMachine.Definition)) {
		return cfg, nil
	}
	// cfg.StateMachine.Definition written definition file path
	var definition jsonRawMessage
	definitionPath := filepath.Clean(filepath.Join(filepath.Dir(path), cfg.StateMachine.Definition))
	log.Println("[debug] definition path =", definitionPath)
	if err := l.load(definitionPath, false, true, &definition); err != nil {
		return nil, fmt.Errorf("load definition `%s`: %w", definitionPath, err)
	}
	cfg.StateMachine.Definition = string(definition)
	return cfg, nil
}

type Config struct {
	RequiredVersion string `yaml:"required_version,omitempty" json:"required_version,omitempty" toml:"required_version,omitempty" env:"REQUIRED_VERSION" validate:"omitempty,version"`
	AWSRegion       string `yaml:"aws_region,omitempty" json:"aws_region,omitempty" toml:"aws_region,omitempty" env:"AWS_REGION" validate:"omitempty,region"`

	StateMachine *StateMachineConfig `yaml:"state_machine,omitempty" json:"state_machine,omitempty"`
	Schedule     []*ScheduleConfig   `yaml:"schedule,omitempty" json:"schedule,omitempty"`
	Tags         map[string]string   `yaml:"tags,omitempty" json:"tags,omitempty"`

	Endpoints *EndpointsConfig `yaml:"endpoints,omitempty" json:"endpoints,omitempty"`

	//private field
	versionConstraints gv.Constraints `yaml:"-,omitempty"`
}

type StateMachineConfig struct {
	Name             string                     `yaml:"name,omitempty" json:"name,omitempty"`
	Type             string                     `yaml:"type,omitempty" json:"type,omitempty"`
	RoleArn          string                     `yaml:"role_arn,omitempty" json:"role_arn,omitempty"`
	Definition       string                     `yaml:"definition,omitempty" json:"definition,omitempty"`
	Logging          *StateMachineLoggingConfig `yaml:"logging,omitempty" json:"logging,omitempty"`
	Tracing          *StateMachineTracingConfig `yaml:"tracing,omitempty" json:"tracing,omitempty"`
	stateMachineType sfntypes.StateMachineType  `yaml:"-,omitempty"`
}

type StateMachineLoggingConfig struct {
	Level                string                                `yaml:"level,omitempty" json:"level,omitempty"`
	IncludeExecutionData *bool                                 `yaml:"include_execution_data,omitempty" json:"include_execution_data,omitempty"`
	Destination          *StateMachineLoggingDestinationConfig `yaml:"destination,omitempty" json:"destination,omitempty"`

	logLevel sfntypes.LogLevel `yaml:"-,omitempty"`
}

type StateMachineLoggingDestinationConfig struct {
	LogGroup string `yaml:"log_group,omitempty" json:"log_group,omitempty"`
}

type StateMachineTracingConfig struct {
	Enabled *bool `yaml:"enabled,omitempty" json:"enabled,omitempty"`
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
	if len(cfg.Schedule) != 0 {
		for i, s := range cfg.Schedule {
			if err := s.Restrict(i, cfg.StateMachine.Name); err != nil {
				return fmt.Errorf("schedule[%d].%w", i, err)
			}
		}
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineConfig) Restrict() error {
	if cfg.Name == "" {
		return errors.New("name is required")
	}
	if cfg.RoleArn == "" {
		return errors.New("role_arn is required")
	}

	var err error
	cfg.stateMachineType, err = restrictSFnStateMachineType(cfg.Type)
	if err != nil {
		return fmt.Errorf("type is %w", err)
	}
	if cfg.Logging == nil {
		return errors.New("logging is required")
	}
	if err := cfg.Logging.Restrict(); err != nil {
		return fmt.Errorf("logging.%w", err)
	}
	if cfg.Tracing == nil {
		return errors.New("tracing is required")
	}
	if err := cfg.Tracing.Restrict(); err != nil {
		return fmt.Errorf("tracing.%w", err)
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineLoggingConfig) Restrict() error {

	if cfg.IncludeExecutionData == nil {
		return errors.New("include_execution_data is required")
	}

	var err error
	cfg.logLevel, err = restrictLogLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("level is %w", err)
	}

	if cfg.Destination == nil {
		if cfg.logLevel != sfntypes.LogLevelOff {
			return errors.New("destination is required, if log_level is not OFF")
		}
	} else {
		if err := cfg.Destination.Restrict(); err != nil {
			return fmt.Errorf("destination.%w", err)
		}
	}

	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineLoggingDestinationConfig) Restrict() error {

	if cfg.LogGroup == "" {
		return errors.New("log_group is required")
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineTracingConfig) Restrict() error {

	if cfg.Enabled == nil {
		return errors.New("enabled is required")
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
			Type:             string(sfntypes.StateMachineTypeStandard),
			stateMachineType: sfntypes.StateMachineTypeStandard,
			Logging: &StateMachineLoggingConfig{
				Level:                string(sfntypes.LogLevelOff),
				logLevel:             sfntypes.LogLevelOff,
				IncludeExecutionData: aws.Bool(true),
			},
			Tracing: &StateMachineTracingConfig{
				Enabled: aws.Bool(false),
			},
		},
		Tags: make(map[string]string),
	}
}

func (cfg *Config) LoadDefinition() (string, error) {
	return cfg.StateMachine.Definition, nil
}

func (cfg *StateMachineConfig) LoadTracingConfiguration() *sfntypes.TracingConfiguration {
	return &sfntypes.TracingConfiguration{
		Enabled: *cfg.Tracing.Enabled,
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
