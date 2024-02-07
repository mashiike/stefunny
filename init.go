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
	AliasName          string `name:"alias" help:"alias name for publish" default:"current" json:"alias,omitempty"`

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
	stateMachine.DeleteTag(tagManagedBy)
	cfg.StateMachine.Value = stateMachine.CreateStateMachineInput
	rules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, stateMachine.QualifiedARN(opt.AliasName))
	if err != nil {
		return err
	}
	if len(rules) > 0 {
		if cfg.Trigger == nil {
			cfg.Trigger = &TriggerConfig{}
		}
		for _, rule := range rules {
			rule.DeleteTag(tagManagedBy)
			eventsRule := TriggerEventConfig{
				KeysToSnakeCase: KeysToSnakeCase[EventBridgeRule]{
					Value:  *rule,
					Strict: true,
				},
			}
			cfg.Trigger.Event = append(cfg.Trigger.Event, eventsRule)
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
