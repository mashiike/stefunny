package stefunny

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	awsarn "github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/google/go-jsonnet/formatter"
	"github.com/mashiike/stefunny/internal/jsonutil"
	"gopkg.in/yaml.v3"
)

type InitInput struct {
	Version            string
	AWSRegion          string
	StateMachineName   string
	ConfigPath         string
	DefinitionFileName string
}

func (app *App) Init(ctx context.Context, input *InitInput) error {
	log.Println("[debug] config path =", input.ConfigPath)
	configDir := filepath.Dir(input.ConfigPath)
	configExt := filepath.Ext(input.ConfigPath)
	if configExt != ".yaml" && configExt != ".yml" {
		return errors.New("config file ext unexpected yaml or yml")
	}
	cfg := NewDefaultConfig()
	if input.Version != "current" && input.Version != "" {
		cfg.RequiredVersion = ">=" + input.Version
	}
	cfg.AWSRegion = input.AWSRegion
	stateMachine, err := app.aws.DescribeStateMachine(ctx, input.StateMachineName)
	if err != nil {
		return fmt.Errorf("failed describe state machine: %w", err)
	}
	cfg.StateMachine = setStateMachineConfig(cfg.StateMachine, stateMachine)

	rules, err := app.aws.SearchScheduleRule(ctx, *stateMachine.StateMachineArn)
	if err != nil {
		return err
	}
	if len(rules) > 0 {
		for _, rule := range rules {
			s, err := newScheduleConfigFromSchedule(rule)
			if err != nil {
				log.Printf("[warn] schedule rule can not managed by %s skip this rule: %s", appName, err)
				continue
			}
			cfg.Schedule = append(cfg.Schedule, s)
		}
	}

	log.Println("[debug] definition path =", input.DefinitionFileName)
	defPath := input.DefinitionFileName
	defPath, err = filepath.Rel(configDir, defPath)
	if err != nil {
		return fmt.Errorf("failed definition path rel: %w", err)
	}
	cfg.StateMachine.Definition = defPath
	defFullPath := filepath.Join(configDir, defPath)
	if err := createDefinitionFile(defFullPath, *stateMachine.Definition); err != nil {
		return fmt.Errorf("failed create definition file: %w", err)
	}
	log.Printf("[notice] StateMachine/%s save state machine definition to %s", *stateMachine.Name, defFullPath)
	if err := createConfigFile(input.ConfigPath, cfg); err != nil {
		return fmt.Errorf("failed create config file: %w", err)
	}
	log.Printf("[notice] StateMachine/%s save config to %s", *stateMachine.Name, input.ConfigPath)
	return nil
}

func setStateMachineConfig(cfg *StateMachineConfig, s *StateMachine) *StateMachineConfig {
	cfg.Name = *s.Name
	cfg.RoleArn = *s.RoleArn
	cfg.Tracing = &StateMachineTracingConfig{
		Enabled: &s.TracingConfiguration.Enabled,
	}
	cfg.Type = string(s.Type)
	cfg.Logging = &StateMachineLoggingConfig{
		Level:                string(s.LoggingConfiguration.Level),
		IncludeExecutionData: &s.LoggingConfiguration.IncludeExecutionData,
	}
	if len(s.LoggingConfiguration.Destinations) > 0 {
		cfg.Logging.Destination = &StateMachineLoggingDestinationConfig{
			LogGroup: extractLogGroupName(*s.LoggingConfiguration.Destinations[0].CloudWatchLogsLogGroup.LogGroupArn),
		}
	}

	return cfg
}

func newScheduleConfigFromSchedule(s *ScheduleRule) (*ScheduleConfig, error) {
	cfg := &ScheduleConfig{}
	cfg.RuleName = coalesceString(s.Name, "")
	cfg.Description = coalesceString(s.Description, "")
	cfg.Expression = *s.ScheduleExpression
	if len(s.Targets) != 1 {
		return nil, fmt.Errorf("rule target must be 1, now %d", len(s.Targets))
	}
	cfg.RoleArn = coalesceString(s.Targets[0].RoleArn, "")
	cfg.ID = coalesceString(s.Targets[0].Id, "")
	return cfg, nil
}

func extractLogGroupName(arn string) string {
	logGroupARN, _ := awsarn.Parse(arn)
	return strings.TrimRight(strings.TrimPrefix(logGroupARN.Resource, "log-group:"), ":*")
}

func createDefinitionFile(path string, definition string) error {
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	switch filepath.Ext(path) {
	case ".json":
		io.WriteString(fp, definition)
	case ".jsonnet":
		formattted, err := formatter.Format(filepath.Base(path), definition, formatter.DefaultOptions())
		if err != nil {
			return err
		}
		io.WriteString(fp, formattted)
	case ".yaml", ".yml":
		bs, err := jsonutil.Json2Yaml([]byte(definition))
		if err != nil {
			return err
		}
		if _, err := fp.Write(bs); err != nil {
			return err
		}
	}
	return nil
}

func createConfigFile(path string, cfg *Config) error {
	fp, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fp.Close()
	bs, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if _, err := fp.Write(bs); err != nil {
		return err
	}
	return nil
}
