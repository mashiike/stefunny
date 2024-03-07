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
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/fujiwara/tfstate-lookup/tfstate"
	"github.com/goccy/go-yaml"
	jsonnet "github.com/google/go-jsonnet"
	gv "github.com/hashicorp/go-version"
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
	funcMap      template.FuncMap
	envs         *OrderdMap[string, string]
	mustEnvs     *OrderdMap[string, string]
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
	for k, v := range funcMap {
		modifiedKey := prefix + k
		if _, ok := l.funcMap[modifiedKey]; ok {
			return fmt.Errorf("funcMap key %s already exists", modifiedKey)
		}
		l.funcMap[modifiedKey] = v
	}
	return nil
}

func (l *ConfigLoader) load(path string, strict bool, withEnv bool, v any) error {
	ext := filepath.Ext(path)
	switch ext {
	case yamlExt, ymlExt:
		b, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if withEnv {
			b, err = l.renderTemplate(b, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
			}
		}
		decoderOpts := []yaml.DecodeOption{
			yaml.UseJSONUnmarshaler(),
		}
		if strict {
			decoderOpts = append(decoderOpts, yaml.DisallowUnknownField())
		}
		dec := yaml.NewDecoder(bytes.NewReader(b), decoderOpts...)
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
			b, err = l.renderTemplate(b, filepath.Dir(path))
			if err != nil {
				return fmt.Errorf("failed to render template: %w", err)
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

func newTemplateFuncEnv(envs *OrderdMap[string, string]) func(string, ...string) string {
	return func(key string, args ...string) string {
		keys := make([]string, 1, len(args))
		keys[0] = key
		var defaultValue string
		if len(args) > 0 {
			// last is default value
			keys = append(keys, args[:len(args)-1]...)
			defaultValue = args[len(args)-1]
		}
		envArgs := key + "," + strings.Join(args, ",")
		for _, k := range keys {
			if v := os.Getenv(k); v != "" {
				envs.Set(envArgs, v)
				return v
			}
		}
		envs.Set(envArgs, defaultValue)
		return defaultValue
	}
}

func newTemplatefuncMustEnv(mustEnvs *OrderdMap[string, string], missingEnvs map[string]struct{}) func(string) string {
	if mustEnvs == nil {
		mustEnvs = NewOrderdMap[string, string]()
	}
	if missingEnvs == nil {
		missingEnvs = make(map[string]struct{})
	}
	return func(key string) string {
		if v, ok := os.LookupEnv(key); ok {
			mustEnvs.Set(key, v)
			return v
		}
		missingEnvs[key] = struct{}{}
		return ""
	}
}

func (l *ConfigLoader) renderTemplate(bs []byte, loadingDir string) ([]byte, error) {
	funcMap := make(template.FuncMap, len(l.funcMap))
	for k, v := range l.funcMap {
		funcMap[k] = v
	}
	l.envs = NewOrderdMap[string, string]()
	l.mustEnvs = NewOrderdMap[string, string]()
	missingEnvs := make(map[string]struct{}, 0)
	if _, ok := funcMap["env"]; !ok {
		funcMap["env"] = newTemplateFuncEnv(l.envs)
	}
	if _, ok := funcMap["must_env"]; !ok {
		funcMap["must_env"] = newTemplatefuncMustEnv(l.mustEnvs, missingEnvs)
	}
	if _, ok := funcMap["json_escape"]; !ok {
		funcMap["json_escape"] = func(v string) (string, error) {
			bs, err := json.Marshal(v)
			if err != nil {
				return "", err
			}
			return string(bs[1 : len(bs)-1]), nil
		}
	}
	if _, ok := funcMap["file"]; !ok {
		funcMap["file"] = func(path string) (string, error) {
			if !filepath.IsAbs(path) {
				path = filepath.Join(loadingDir, path)
			}
			bs, err := os.ReadFile(path)
			if err != nil {
				return "", err
			}
			return string(bs), nil
		}
	}
	tmpl, err := template.New("config").Funcs(funcMap).Parse(string(bs))
	if err != nil {
		return nil, fmt.Errorf("template parse error: %w", err)
	}
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, nil); err != nil {
		return nil, fmt.Errorf("template execute error: %w", err)
	}
	if len(missingEnvs) > 0 {
		for k := range missingEnvs {
			log.Printf("[warn] environment variable `%s` is not defined", k)
		}
		return nil, fmt.Errorf("missing %d environment variables", len(missingEnvs))
	}
	return buf.Bytes(), nil
}

func (l *ConfigLoader) Load(ctx context.Context, path string) (*Config, error) {
	cfg := NewDefaultConfig()
	if err := l.setConfigPath(cfg, path); err != nil {
		return nil, fmt.Errorf("set config path:%w", err)
	}
	if err := l.preLoadForTemplateFuncs(ctx, cfg, path); err != nil {
		return nil, fmt.Errorf("pre load for template funcs: %w", err)
	}
	if cfg.StateMachine == nil {
		cfg.StateMachine = &StateMachineConfig{}
	}
	cfg.StateMachine.Strict = true
	if err := l.load(path, true, true, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	if err := l.migrationForDeprecatedFields(ctx, cfg); err != nil {
		return nil, fmt.Errorf("migration for deprecated fields: %w", err)
	}
	cfg.Envs = l.envs
	cfg.MustEnvs = l.mustEnvs
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
	var definition json.RawMessage
	definitionPath := filepath.Clean(filepath.Join(filepath.Dir(path), cfg.StateMachine.DefinitionPath))
	log.Println("[debug] definition path =", definitionPath)
	if err := l.load(definitionPath, false, true, &definition); err != nil {
		return nil, fmt.Errorf("load definition `%s`: %w", definitionPath, err)
	}
	cfg.StateMachine.Value.Definition = aws.String(string(definition))
	return cfg, nil
}

func (l *ConfigLoader) setConfigPath(cfg *Config, path string) error {
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("[debug] os.Getwd: %s", err)
		return err
	}

	if !filepath.IsAbs(dir) {
		dir, err = filepath.Abs(dir)
		if err != nil {
			log.Printf("[debug] filepath.Abs: %s", err)
			return err
		}
	}
	pathDir := filepath.Dir(path)
	if !filepath.IsAbs(pathDir) {
		pathDir, err = filepath.Abs(pathDir)
		if err != nil {
			log.Printf("[debug] filepath.Abs: %s", err)
			return err
		}
	}
	relPath, err := filepath.Rel(dir, pathDir)
	if err != nil {
		log.Printf("[debug] filepath.Rel: %s", err)
		return err
	}

	cfg.ConfigDir = relPath
	cfg.ConfigFileName = filepath.Base(path)
	return nil
}

// pre load for tfstate path read
func (l *ConfigLoader) preLoadForTemplateFuncs(ctx context.Context, cfg *Config, path string) error {
	if err := l.load(path, false, false, cfg); err != nil {
		return err
	}
	bs, err := json.Marshal(cfg.TFState)
	if err != nil {
		return fmt.Errorf("tfstate marshal:%w", err)
	}
	bs, err = l.renderTemplate(bs, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("render template:%w", err)
	}
	if err := json.Unmarshal(bs, &cfg.TFState); err != nil {
		return fmt.Errorf("tfstate unmarshal:%w", err)
	}
	for i, tfstate := range cfg.TFState {
		if tfstate.Location == "" {
			return fmt.Errorf("tfstate[%d].location is required", i)
		}
		loc := tfstate.Location
		u, err := url.Parse(loc)
		if err != nil || (u != nil && u.Scheme == "") {
			loc = filepath.Join(filepath.Dir(path), loc)
		}
		if err := l.AppendTFState(ctx, tfstate.FuncPrefix, loc); err != nil {
			return fmt.Errorf("tfstate[%d] %w", i, err)
		}
	}
	return nil
}

func (l *ConfigLoader) migrationForDeprecatedFields(ctx context.Context, cfg *Config) error {
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
						return fmt.Errorf("load aws config:%w", err)
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
						return fmt.Errorf("describe log groups:%w", err)
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
					return fmt.Errorf("log group `%s` not found", cfg.StateMachine.Logging.Destination.LogGroup)
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
				KeysToSnakeCase: NewKeysToSnakeCase(TriggerEventConfigInner{
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
	return nil
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

	ConfigDir      string                     `yaml:"-" json:"-"`
	ConfigFileName string                     `yaml:"-" json:"-"`
	Envs           *OrderdMap[string, string] `yaml:"-" json:"-"`
	MustEnvs       *OrderdMap[string, string] `yaml:"-" json:"-"`
	//private field
	mu                 sync.Mutex
	versionConstraints gv.Constraints `yaml:"-,omitempty"`
	awsCfg             *aws.Config    `yaml:"-"`
}

type TFStateConfig struct {
	FuncPrefix string `yaml:"func_prefix,omitempty" json:"func_prefix,omitempty"`
	Location   string `yaml:"location,omitempty" json:"location,omitempty"`
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
	KeysToSnakeCase[scheduler.CreateScheduleInput] `yaml:",inline" json:",inline"`
}

type TriggerEventConfig struct {
	KeysToSnakeCase[TriggerEventConfigInner] `yaml:",inline" json:",inline"`
}

type TriggerEventConfigInner struct {
	eventbridge.PutRuleInput `yaml:",inline"`
	Target                   eventbridgetypes.Target `yaml:"Target,omitempty" json:"Target,omitempty"`
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
		if err := cfg.Trigger.Restrict(cfg.StateMachineName()); err != nil {
			return fmt.Errorf("trigger.%w", err)
		}
	}
	if len(cfg.Tags) > 0 {
		log.Println("[warn] tags is deprecated. Use state_machine.tags instead. (since v0.6.0)")
	}
	return nil
}

func (cfg *TriggerConfig) Restrict(stateMachineName string) error {
	for i, s := range cfg.Schedule {
		if err := s.Restrict(i, stateMachineName); err != nil {
			return fmt.Errorf("schedule[%d].%w", i, err)
		}
	}
	for i, e := range cfg.Event {
		if err := e.Restrict(i, stateMachineName); err != nil {
			return fmt.Errorf("event[%d].%w", i, err)
		}
	}
	return nil
}

func (cfg *TriggerEventConfig) Restrict(i int, stateMachineName string) error {
	cfg.Value.PutRuleInput.RoleArn = ptr(coalesce(cfg.Value.PutRuleInput.RoleArn, cfg.Value.Target.RoleArn))
	cfg.Value.Target.RoleArn = nil
	if coalesce(cfg.Value.PutRuleInput.RoleArn) == "" && coalesce(cfg.Value.Target.RoleArn) == "" {
		return errors.New("role_arn is required")
	}
	if coalesce(cfg.Value.PutRuleInput.Name) == "" {
		log.Printf("[warn] trigger.event[%d].rule_name is empty. Use state_machine.name as rule_name.", i)
		cfg.Value.PutRuleInput.Name = aws.String(stateMachineName)
	}
	if coalesce(cfg.Value.PutRuleInput.ScheduleExpression) == "" && coalesce(cfg.Value.PutRuleInput.EventPattern) == "" {
		return errors.New("schedule_expression or event_pattern is required")
	}
	if cfg.Value.Target.Arn != nil {
		return errors.New("target.arn is not allowed")
	}
	if cfg.Value.State == "" {
		cfg.Value.State = eventbridgetypes.RuleStateEnabled
	}
	return nil
}

func (cfg *TriggerEventConfig) UnmarshalJSON(b []byte) error {
	cfg.Strict = true
	if err := json.Unmarshal(b, &cfg.KeysToSnakeCase); err != nil {
		return err
	}
	return nil
}

func (cfg *TriggerScheduleConfig) Restrict(i int, stateMachineName string) error {
	if coalesce(cfg.Value.Name) == "" {
		log.Printf("[warn] trigger.schedule[%d].schedule_name is empty. Use state_machine.name as rule_name.", i)
		cfg.Value.Name = aws.String(stateMachineName)
	}
	if coalesce(cfg.Value.ScheduleExpression) == "" {
		return errors.New("schedule_expression is required")
	}
	if cfg.Value.Target == nil {
		return nil
	}
	if coalesce(cfg.Value.Target.Arn) != "" {
		return errors.New("target.arn is not allowed")
	}
	return nil
}

func (cfg *TriggerScheduleConfig) UnmarshalJSON(b []byte) error {
	cfg.Strict = true
	if err := json.Unmarshal(b, &cfg.KeysToSnakeCase); err != nil {
		return err
	}
	return nil
}

func (cfg *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	cfg.mu.Lock()
	defer cfg.mu.Unlock()
	if cfg.awsCfg != nil {
		return *cfg.awsCfg, nil
	}
	opts := []func(*awsconfig.LoadOptions) error{}
	if cfg.AWSRegion != "" {
		log.Printf("[debug] use aws_region = %s", cfg.AWSRegion)
		opts = append(opts, awsconfig.WithRegion(cfg.AWSRegion))
	}
	if endpointsResolver, ok := cfg.EndpointResolver(); ok {
		opts = append(opts, awsconfig.WithEndpointResolverWithOptions(endpointsResolver))
	}
	log.Println("[debug] load aws default config")
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}
	stsClient := sts.NewFromConfig(awsCfg)
	identity, err := stsClient.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err == nil {
		log.Printf("[debug] caller identity: %s", *identity.Arn)
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

func (cfg *Config) NewStateMachine() *StateMachine {
	stateMachine := &StateMachine{
		CreateStateMachineInput: cfg.StateMachine.Value,
	}
	stateMachine.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	stateMachine.AppendTags(cfg.Tags)

	stateMachine.ConfigFilePath = aws.String(filepath.Join(cfg.ConfigDir, cfg.ConfigFileName))
	stateMachine.DefinitionPath = aws.String(filepath.Join(cfg.ConfigDir, cfg.StateMachine.DefinitionPath))
	return stateMachine
}

func (cfg *Config) NewEventBridgeRules() EventBridgeRules {
	if cfg.Trigger == nil {
		return EventBridgeRules{}
	}
	tags := make(map[string]string)
	for _, tag := range cfg.StateMachine.Value.Tags {
		tags[coalesce(tag.Key)] = coalesce(tag.Value)
	}
	for k, v := range cfg.Tags {
		tags[k] = v
	}
	tags[tagManagedBy] = appName
	rules := make(EventBridgeRules, 0, len(cfg.Trigger.Event))
	for _, e := range cfg.Trigger.Event {
		rule := &EventBridgeRule{
			PutRuleInput:   e.Value.PutRuleInput,
			Target:         e.Value.Target,
			ConfigFilePath: aws.String(filepath.Join(cfg.ConfigDir, cfg.ConfigFileName)),
		}
		if rule.Target.RoleArn == nil && e.Value.RoleArn != nil {
			rule.Target.RoleArn = e.Value.RoleArn
		}
		if rule.RoleArn == nil && e.Value.Target.RoleArn != nil {
			rule.RoleArn = e.Value.Target.RoleArn
		}
		rule.AppendTags(tags)
		rules = append(rules, rule)
	}
	sort.Sort(rules)
	return rules
}

func (cfg *Config) NewSchedules() Schedules {
	if cfg.Trigger == nil {
		return Schedules{}
	}
	schedules := make(Schedules, 0, len(cfg.Trigger.Schedule))
	for _, s := range cfg.Trigger.Schedule {
		schedule := &Schedule{
			CreateScheduleInput: s.Value,
			ConfigFilePath:      aws.String(filepath.Join(cfg.ConfigDir, cfg.ConfigFileName)),
		}
		if schedule.HasItPassed() {
			log.Printf("[warn] schedule %s has passed, ignore this schedule", coalesce(schedule.Name))
			continue
		}
		schedules = append(schedules, schedule)
	}
	sort.Sort(schedules)
	return schedules
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
		logGroupArn, err := arn.Parse(*dest.CloudWatchLogsLogGroup.LogGroupArn)
		if err != nil {
			return fmt.Errorf(
				"logging_configuration.destinations[%d].cloudwatch_logs_log_group.log_group_arn = `%s` is invalid: %w",
				i, *dest.CloudWatchLogsLogGroup.LogGroupArn, err,
			)
		}
		if logGroupArn.Service != "logs" {
			return fmt.Errorf("logging_configuration.destinations[%d].cloudwatch_logs_log_group.log_group_arn is not CloudWatch Logs Arn", i)
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
		Tags:     make(map[string]string),
		Envs:     NewOrderdMap[string, string](),
		MustEnvs: NewOrderdMap[string, string](),
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
