package stefunny

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
)

type InitOption struct {
	StateMachineName   string   `name:"state-machine" help:"AWS StepFunctions state machine name" required:"" env:"STATE_MACHINE_NAME" json:"state_machine_name,omitempty"`
	DefinitionFilePath string   `name:"definition" short:"d" help:"Path to state machine definition file" default:"definition.asl.json" type:"path" env:"DEFINITION_FILE_PATH" json:"definition_file_path,omitempty"`
	TFState            string   `kong:"-" help:"Path to terraform state file" type:"path" json:"tfstate,omitempty"` // TODO: if removed global flag, not ignore this flag for kong
	Envs               []string `name:"env" help:"templateize environment variables" json:"envs,omitempty"`
	MustEnvs           []string `name:"must-env" help:"templateize must environment variables" json:"must_envs,omitempty"`
	SkipTrigger        bool     `name:"skip-trigger" help:"Skip trigger" json:"skip_trigger,omitempty"`
	ConfigPath         string   `kong:"-" json:"-"`
	AWSRegion          string   `kong:"-" json:"-"`
}

func (app *App) Init(ctx context.Context, opt InitOption) error {
	log.Println("[debug] config path =", opt.ConfigPath)
	log.Println("[debug] definition path =", opt.DefinitionFilePath)

	configDir := filepath.Dir(opt.ConfigPath)
	defPath := opt.DefinitionFilePath
	var err error
	defPath, err = filepath.Rel(configDir, defPath)
	if err != nil {
		return fmt.Errorf("failed definition path rel: %w", err)
	}

	cfg, err := app.makeConfig(ctx, defPath, opt.SkipTrigger, &DescribeStateMachineInput{
		Name: opt.StateMachineName,
	})
	if err != nil {
		return fmt.Errorf("failed to make config: %w", err)
	}
	if opt.AWSRegion != "" {
		cfg.AWSRegion = opt.AWSRegion
	}

	templateize, err := prepareForTamplatize(cfg, opt.TFState, opt.Envs, opt.MustEnvs)
	if err != nil {
		return fmt.Errorf("failed prepare for templateize: %w", err)
	}

	renderer := NewRenderer(cfg)
	log.Printf("[notice] StateMachine/%s save config to %s", opt.StateMachineName, opt.ConfigPath)
	if err := renderer.CreateConfigFile(ctx, opt.ConfigPath, templateize); err != nil {
		return fmt.Errorf("failed create config file: %w", err)
	}

	defFullPath := filepath.Join(configDir, defPath)
	log.Printf("[notice] StateMachine/%s save state machine definition to %s", opt.StateMachineName, defFullPath)
	if err := renderer.CreateDefinitionFile(ctx, defFullPath, templateize); err != nil {
		return fmt.Errorf("failed create state machine definition file: %w", err)
	}
	return nil
}

func (app *App) makeConfig(ctx context.Context, defPath string, skipTrigger bool, params *DescribeStateMachineInput) (*Config, error) {
	cfg := NewDefaultConfig()
	cfg.RequiredVersion = ">=" + Version

	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to describe state machine: %w", err)
	}
	stateMachine.DeleteTag(tagManagedBy)
	cfg.StateMachine.Value = stateMachine.CreateStateMachineInput
	cfg.StateMachine.SetDetinitionPath(defPath)
	cfg.StateMachine.SetDefinition(coalesce(stateMachine.Definition))
	if skipTrigger {
		return cfg, nil
	}
	triggerCfg, err := app.makeTrigerConfig(ctx, stateMachine)
	if err != nil {
		return nil, fmt.Errorf("failed to make trigger config: %w", err)
	}
	cfg.Trigger = triggerCfg
	return cfg, nil
}

func (app *App) makeTrigerConfig(ctx context.Context, stateMachine *StateMachine) (*TriggerConfig, error) {
	rules, err := app.eventbridgeSvc.SearchRelatedRules(ctx, &SearchRelatedRulesInput{
		StateMachineQualifiedArn: stateMachine.QualifiedArn(app.StateMachineAliasName()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed search related rules: %w", err)
	}
	schedules, err := app.schedulerSvc.SearchRelatedSchedules(ctx, &SearchRelatedSchedulesInput{
		StateMachineQualifiedArn: stateMachine.QualifiedArn(app.StateMachineAliasName()),
	})
	if err != nil {
		return nil, fmt.Errorf("failed search related schedules: %w", err)
	}
	trigger := &TriggerConfig{}
	if len(rules) > 0 {
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
			eventsRule.Value.State = ""
			if eventsRule.Value.Target.RoleArn == nil && eventsRule.Value.RoleArn != nil {
				eventsRule.Value.Target.RoleArn = eventsRule.Value.RoleArn
				eventsRule.Value.RoleArn = nil
			}
			if coalesce(eventsRule.Value.Target.RoleArn) == coalesce(eventsRule.Value.RoleArn) {
				eventsRule.Value.RoleArn = nil
			}
			if len(rule.AdditionalTargets) > 0 {
				log.Printf("[debug] StateMachine/%s has additional targets, skip non related target", coalesce(stateMachine.Name))
			}
			trigger.Event = append(trigger.Event, eventsRule)
		}
	}
	if len(schedules) > 0 {
		for _, schedule := range schedules {
			if schedule.Target != nil {
				schedule.Target.Arn = nil
			}
			scheduleRule := TriggerScheduleConfig{
				KeysToSnakeCase: KeysToSnakeCase[scheduler.CreateScheduleInput]{
					Value:  schedule.CreateScheduleInput,
					Strict: true,
				},
			}
			scheduleRule.Value.State = ""
			trigger.Schedule = append(trigger.Schedule, scheduleRule)
		}
	}
	return trigger, nil
}

func prepareForTamplatize(cfg *Config, tfstateLoc string, envs []string, mustEnvs []string) (bool, error) {
	var templateize bool
	if len(envs) > 0 {
		envsMap := NewOrderdMap[string, string]()
		envFunc := newTemplateFuncEnv(envsMap)
		for _, env := range envs {
			envFunc(env)
		}
		cfg.Envs = envsMap
		templateize = true
	}
	if len(mustEnvs) > 0 {
		mustEnvsMap := NewOrderdMap[string, string]()
		missingEnvs := make(map[string]struct{})
		mustEnvFunc := newTemplatefuncMustEnv(mustEnvsMap, missingEnvs)
		for _, env := range mustEnvs {
			mustEnvFunc(env)
		}
		if len(missingEnvs) > 0 {
			for env := range missingEnvs {
				log.Printf("[warn] environment variable `%s` is not defined", env)
			}
			return false, fmt.Errorf("%d environment variables are not defined", len(missingEnvs))
		}
		cfg.MustEnvs = mustEnvsMap
		templateize = true
	}
	if tfstateLoc != "" {
		cfg.TFState = []*TFStateConfig{
			{
				Location: tfstateLoc,
			},
		}
		templateize = true
	}
	return templateize, nil
}
