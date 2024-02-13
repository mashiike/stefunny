package stefunny

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
)

type PullOption struct {
	Templateize bool   `name:"templateize" default:"true" negatable:"" help:"templateize output"`
	Qualifier   string `name:"qualifier" help:"qualifier for the version"`
}

func (app *App) Pull(ctx context.Context, opt PullOption) error {
	defPath := filepath.Join(app.cfg.ConfigDir, app.cfg.StateMachine.DefinitionPath)
	cfg, err := app.makeConfig(ctx, defPath, true, &DescribeStateMachineInput{
		Name:      app.cfg.StateMachineName(),
		Qualifier: opt.Qualifier,
	})
	if err != nil {
		return fmt.Errorf("failed to make config: %w", err)
	}
	cfg.Envs = app.cfg.Envs
	cfg.MustEnvs = app.cfg.MustEnvs
	cfg.TFState = app.cfg.TFState
	renderer := NewRenderer(cfg)
	if err := renderer.CreateDefinitionFile(ctx, defPath, opt.Templateize); err != nil {
		return fmt.Errorf("failed create state machine definition file: %w", err)
	}
	log.Printf("[notice] StateMachine/%s save state machine definition to %s", app.cfg.StateMachineName(), defPath)
	return nil
}
