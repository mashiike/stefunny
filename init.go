package stefunny

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
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
	cfg := NewDefaultConfig()
	cfg.RequiredVersion = ">=" + Version
	cfg.AWSRegion = opt.AWSRegion
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, opt.StateMachineName)
	if err != nil {
		return fmt.Errorf("failed describe state machine: %w", err)
	}
	cfg.StateMachine = setStateMachineConfig(cfg.StateMachine, stateMachine)

	rules, err := app.eventbridgeSvc.SearchScheduleRule(ctx, coalesce(stateMachine.StateMachineArn))
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
	cfg.StateMachine.SetDefinition(coalesce(stateMachine.Definition))
	renderer := NewRenderer(cfg)
	log.Printf("[notice] StateMachine/%s save config to %s", coalesce(stateMachine.Name), opt.ConfigPath)
	if err := renderer.RenderConfigFile(opt.ConfigPath); err != nil {
		return fmt.Errorf("failed render config file: %w", err)
	}

	defFullPath := filepath.Join(configDir, defPath)
	log.Printf("[notice] StateMachine/%s save state machine definition to %s", coalesce(stateMachine.Name), defFullPath)
	if err := renderer.RenderDefinitionFile(defFullPath); err != nil {
		return fmt.Errorf("failed render state machine definition file: %w", err)
	}
	return nil
}

func setStateMachineConfig(cfg *StateMachineConfig, s *StateMachine) *StateMachineConfig {
	cfg.Value = s.CreateStateMachineInput
	for i := 0; i < len(cfg.Value.Tags); i++ {
		tag := cfg.Value.Tags[i]
		if *tag.Key == tagManagedBy {
			cfg.Value.Tags = append(cfg.Value.Tags[:i], cfg.Value.Tags[i+1:]...)
			continue
		}
	}

	return cfg
}

func newScheduleConfigFromSchedule(s *ScheduleRule) (*ScheduleConfig, error) {
	cfg := &ScheduleConfig{}
	cfg.RuleName = coalesce(s.Name)
	cfg.Description = coalesce(s.Description)
	cfg.Expression = coalesce(s.ScheduleExpression)
	if len(s.Targets) != 1 {
		return nil, fmt.Errorf("rule target must be 1, now %d", len(s.Targets))
	}
	cfg.RoleArn = coalesce(s.Targets[0].RoleArn)
	cfg.ID = coalesce(s.Targets[0].Id)
	return cfg, nil
}
