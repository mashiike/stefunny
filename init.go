package stefunny

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
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
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name: opt.StateMachineName,
	})
	if err != nil {
		return fmt.Errorf("failed describe state machine: %w", err)
	}
	stateMachine.DeleteTag(tagManagedBy)
	cfg.StateMachine.Value = stateMachine.CreateStateMachineInput
	rules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, &SearchRelatedRulesInput{
		StateMachineQualifiedARN: stateMachine.QualifiedARN(app.StateMachineAliasName()),
	})
	if err != nil {
		return fmt.Errorf("failed search related rules: %w", err)
	}
	if len(rules) > 0 {
		if cfg.Trigger == nil {
			cfg.Trigger = &TriggerConfig{}
		}
		for _, rule := range rules {
			rule.DeleteTag(tagManagedBy)
			rule.Target.Arn = nil
			eventsRule := TriggerEventConfig{
				KeysToSnakeCase: KeysToSnakeCase[TriggerEventConfigInner]{
					Value: TriggerEventConfigInner{
						PutRuleInput: rule.PutRuleInput,
						Target:       rule.Target,
					},
					Strict: true,
				},
			}
			if len(rule.AdditionalTargets) > 0 {
				log.Printf("[debug] StateMachine/%s has additional targets, skip non related target", coalesce(stateMachine.Name))
			}
			cfg.Trigger.Event = append(cfg.Trigger.Event, eventsRule)
		}
	}

	schedules, err := app.schedulerSvc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
		StateMachineQualifiedARN: stateMachine.QualifiedARN(app.StateMachineAliasName()),
	})
	if err != nil {
		return fmt.Errorf("failed search related schedules: %w", err)
	}
	if len(schedules) > 0 {
		if cfg.Trigger == nil {
			cfg.Trigger = &TriggerConfig{}
		}
		for _, schedule := range schedules {
			if schedule.CreateScheduleInput.Target != nil {
				schedule.CreateScheduleInput.Target.Arn = nil
			}
			scheduleRule := TriggerScheduleConfig{
				KeysToSnakeCase: KeysToSnakeCase[scheduler.CreateScheduleInput]{
					Value:  schedule.CreateScheduleInput,
					Strict: true,
				},
			}
			cfg.Trigger.Schedule = append(cfg.Trigger.Schedule, scheduleRule)
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
