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

	"github.com/google/go-jsonnet/formatter"
	"github.com/mashiike/stefunny/internal/jsonutil"
	"gopkg.in/yaml.v3"
)

type InitInput struct {
	Version            string
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
	stateMachine, err := app.aws.DescribeStateMachine(ctx, input.StateMachineName)
	if err != nil {
		return fmt.Errorf("failed describe state machine: %w", err)
	}
	cfg.StateMachine = setStateMachineConfig(cfg.StateMachine, stateMachine)

	log.Println("[debug] definition path =", input.DefinitionFileName)
	defPath := input.DefinitionFileName
	if filepath.IsAbs(defPath) {
		absConfigDir, err := filepath.Abs(configDir)
		if err != nil {
			return fmt.Errorf("failed get abs path rel: %w", err)
		}
		defPath, err = filepath.Rel(absConfigDir, defPath)
		if err != nil {
			return fmt.Errorf("failed definition path rel: %w", err)
		}
	}
	cfg.StateMachine.Definition = defPath
	if err := createDefinitionFile(filepath.Join(configDir, defPath), *stateMachine.Definition); err != nil {
		return fmt.Errorf("failed create definition file: %w", err)
	}
	if err := createConfigFile(input.ConfigPath, cfg); err != nil {
		return fmt.Errorf("failed create config file: %w", err)
	}
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

func extractLogGroupName(arn string) string {
	parts := strings.Split(arn, "/")
	return parts[len(parts)-1]
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
