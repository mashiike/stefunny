package stefunny

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type InitOption struct {
	StateMachineName   string `name:"state-machine" help:"AWS StepFunctions state machine name" required:"" env:"STATE_MACHINE_NAME" json:"state_machine_name,omitempty"`
	DefinitionFilePath string `name:"definition" short:"d" help:"Path to state machine definition file" default:"definition.asl.json" type:"path" env:"DEFINITION_FILE_PATH" json:"definition_file_path,omitempty"`

	ConfigPath string `kong:"-" json:"-"`
	AWSRegion  string `kong:"-" json:"-"`
}

func (app *App) Init(ctx context.Context, opt InitOption) error {
	log.Println("[debug] config path =", opt.ConfigPath)
	configDir := filepath.Dir(opt.ConfigPath)
	configExt := filepath.Ext(opt.ConfigPath)
	if configExt != ".yaml" && configExt != ".yml" {
		return errors.New("config file ext unexpected yaml or yml")
	}
	cfg := NewDefaultConfig()
	cfg.RequiredVersion = ">=" + Version
	cfg.AWSRegion = opt.AWSRegion
	stateMachine, err := app.aws.DescribeStateMachine(ctx, opt.StateMachineName)
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

	log.Println("[debug] definition path =", opt.DefinitionFilePath)
	defPath := opt.DefinitionFilePath
	defPath, err = filepath.Rel(configDir, defPath)
	if err != nil {
		return fmt.Errorf("failed definition path rel: %w", err)
	}
	cfg.StateMachine.SetDetinitionPath(defPath)
	defFullPath := filepath.Join(configDir, defPath)
	if err := createDefinitionFile(defFullPath, *stateMachine.Definition); err != nil {
		return fmt.Errorf("failed create definition file: %w", err)
	}
	log.Printf("[notice] StateMachine/%s save state machine definition to %s", *stateMachine.Name, defFullPath)
	if err := createConfigFile(opt.ConfigPath, cfg); err != nil {
		return fmt.Errorf("failed create config file: %w", err)
	}
	log.Printf("[notice] StateMachine/%s save config to %s", *stateMachine.Name, opt.ConfigPath)
	return nil
}

func setStateMachineConfig(cfg *StateMachineConfig, s *StateMachine) *StateMachineConfig {
	cfg.KeysToSnakeCase.Value = s.CreateStateMachineInput
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
		formatted, err := JSON2Jsonnet(filepath.Base(path), []byte(definition))
		if err != nil {
			return err
		}
		if _, err := fp.Write(formatted); err != nil {
			return err
		}
	case ".yaml", ".yml":
		bs, err := JSON2YAML([]byte(definition))
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
