package stefunny

import (
	"context"
	"fmt"
	"log"
	"sync"
)

const (
	tagManagedBy     = "ManagedBy"
	appName          = "stefunny"
	defaultAliasName = "current"
)

type App struct {
	mu             sync.Mutex
	cfg            *Config
	sfnSvc         SFnService
	eventbridgeSvc EventBridgeService
	schedulerSvc   SchedulerService
	aliasName      string
}

type NewAppOption func(*App)

func (app *App) sfnService(ctx context.Context) (SFnService, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.sfnSvc != nil {
		return app.sfnSvc, nil
	}
	awsCfg, err := app.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SFN client: %w", err)
	}
	client := app.cfg.NewStepFunctionsClientFromConfig(awsCfg)
	svc := NewSFnService(client)
	svc.SetAliasName(app.aliasName)
	app.sfnSvc = svc
	return app.sfnSvc, nil
}

func (app *App) eventBridgeService(ctx context.Context) (EventBridgeService, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.eventbridgeSvc != nil {
		return app.eventbridgeSvc, nil
	}
	awsCfg, err := app.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get EventBridge client: %w", err)
	}
	client := app.cfg.NewEventBridgeClientFromConfig(awsCfg)
	app.eventbridgeSvc = NewEventBridgeService(client)
	return app.eventbridgeSvc, nil
}

func (app *App) schedulerService(ctx context.Context) (SchedulerService, error) {
	app.mu.Lock()
	defer app.mu.Unlock()
	if app.schedulerSvc != nil {
		return app.schedulerSvc, nil
	}
	awsCfg, err := app.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Scheduler client: %w", err)
	}
	client := app.cfg.NewSchedulerClientFromConfig(awsCfg)
	app.schedulerSvc = NewSchedulerService(client)
	return app.schedulerSvc, nil
}

// WithSFNClient sets the SFn client for New(ctx, cfg, opts...)
// this is for testing
func WithSFnClient(sfnClient SFnClient) NewAppOption {
	return func(app *App) {
		app.sfnSvc = NewSFnService(sfnClient)
	}
}

// WithSFnService sets the SFn service for New(ctx, cfg, opts...)
func WithSFnService(sfnService SFnService) NewAppOption {
	return func(app *App) {
		app.sfnSvc = sfnService
	}
}

func WithSchedulerService(schedulerService SchedulerService) NewAppOption {
	return func(app *App) {
		app.schedulerSvc = schedulerService
	}
}

func WithSchedulerClient(schedulerClient SchedulerClient) NewAppOption {
	return func(app *App) {
		app.schedulerSvc = NewSchedulerService(schedulerClient)
	}
}

// WithEventBridgeClient sets the EventBridge client for New(ctx, cfg, opts...)
// this is for testing
func WithEventBridgeClient(eventBridgeClient EventBridgeClient) NewAppOption {
	return func(app *App) {
		app.eventbridgeSvc = NewEventBridgeService(eventBridgeClient)
	}
}

// WithEventBridgeService sets the EventBridge service for New(ctx, cfg, opts...)
func WithEventBridgeService(eventBridgeService EventBridgeService) NewAppOption {
	return func(app *App) {
		app.eventbridgeSvc = eventBridgeService
	}
}

func (app *App) SetAliasName(aliasName string) {
	if aliasName == "" {
		aliasName = defaultAliasName
	}
	app.mu.Lock()
	defer app.mu.Unlock()
	app.aliasName = aliasName
	if app.sfnSvc != nil {
		app.sfnSvc.SetAliasName(aliasName)
	}
	log.Printf("[debug] set state machine alias name %s", aliasName)
}

func (app *App) StateMachineAliasName() string {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.aliasName
}

// New creates a new App
func New(_ context.Context, cfg *Config, opts ...NewAppOption) (*App, error) {
	app := &App{
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(app)
	}
	app.SetAliasName("")
	return app, nil
}
