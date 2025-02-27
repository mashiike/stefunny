package stefunny

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
)

const (
	tagManagedBy     = "ManagedBy"
	appName          = "stefunny"
	defaultAliasName = "current"
)

type App struct {
	cfg            *Config
	sfnSvc         SFnService
	eventbridgeSvc EventBridgeService
	schedulerSvc   SchedulerService
	aliasName      string
}

type newAppOptions struct {
	mu             sync.Mutex
	cfg            *Config
	sfnSvc         SFnService
	eventbridgeSvc EventBridgeService
	schedulerSvc   SchedulerService
	awsCfg         *aws.Config
}

type NewAppOption func(*newAppOptions)

func (o *newAppOptions) GetSFnService(ctx context.Context) (SFnService, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.sfnSvc != nil {
		return o.sfnSvc, nil
	}
	awsCfg, err := o.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := o.cfg.NewStepFunctionsClientFromConfig(awsCfg)
	o.sfnSvc = NewSFnService(client)
	return o.sfnSvc, nil
}

func (o *newAppOptions) GetEventBridgeService(ctx context.Context) (EventBridgeService, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.eventbridgeSvc != nil {
		return o.eventbridgeSvc, nil
	}
	awsCfg, err := o.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := o.cfg.NewEventBridgeClientFromConfig(awsCfg)
	o.eventbridgeSvc = NewEventBridgeService(client)
	return o.eventbridgeSvc, nil
}

func (o *newAppOptions) GetSchedulerService(ctx context.Context) (SchedulerService, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.schedulerSvc != nil {
		return o.schedulerSvc, nil
	}
	awsCfg, err := o.cfg.LoadAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := o.cfg.NewSchedulerClientFromConfig(awsCfg)
	o.schedulerSvc = NewSchedulerService(client)
	return o.schedulerSvc, nil
}

// WithSFNClient sets the SFn client for New(ctx, cfg, opts...)
// this is for testing
func WithSFnClient(sfnClient SFnClient) NewAppOption {
	return func(o *newAppOptions) {
		o.sfnSvc = NewSFnService(sfnClient)
	}
}

// WithSFnService sets the SFn service for New(ctx, cfg, opts...)
func WithSFnService(sfnService SFnService) NewAppOption {
	return func(o *newAppOptions) {
		o.sfnSvc = sfnService
	}
}

func WithSchedulerService(schedulerService SchedulerService) NewAppOption {
	return func(o *newAppOptions) {
		o.schedulerSvc = schedulerService
	}
}

func WithSchedulerClient(schedulerClient SchedulerClient) NewAppOption {
	return func(o *newAppOptions) {
		o.schedulerSvc = NewSchedulerService(schedulerClient)
	}
}

// WithEventBridgeClient sets the EventBridge client for New(ctx, cfg, opts...)
// this is for testing
func WithEventBridgeClient(eventBridgeClient EventBridgeClient) NewAppOption {
	return func(o *newAppOptions) {
		o.eventbridgeSvc = NewEventBridgeService(eventBridgeClient)
	}
}

// WithEventBridgeService sets the EventBridge service for New(ctx, cfg, opts...)
func WithEventBridgeService(eventBridgeService EventBridgeService) NewAppOption {
	return func(o *newAppOptions) {
		o.eventbridgeSvc = eventBridgeService
	}
}

// WithAWSConfig sets the AWS config for New(ctx, cfg, opts...)
// this is for testing
func WithAWSConfig(awsCfg aws.Config) NewAppOption {
	return func(o *newAppOptions) {
		o.awsCfg = &awsCfg
	}
}

func (app *App) SetAliasName(aliasName string) {
	if aliasName == "" {
		aliasName = defaultAliasName
	}
	app.aliasName = aliasName
	app.sfnSvc.SetAliasName(aliasName)
	log.Printf("[debug] set state machine alias name %s", aliasName)
}

func (app *App) StateMachineAliasName() string {
	return app.aliasName
}

// New creates a new App
func New(ctx context.Context, cfg *Config, opts ...NewAppOption) (*App, error) {
	o := newAppOptions{
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(&o)
	}
	sfnSvc, err := o.GetSFnService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get SFN client: %w", err)
	}
	eventbridgeSvc, err := o.GetEventBridgeService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get EventBridge client: %w", err)
	}
	scheduelrSvc, err := o.GetSchedulerService(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Scheduler client: %w", err)
	}
	app := &App{
		cfg:            cfg,
		sfnSvc:         sfnSvc,
		eventbridgeSvc: eventbridgeSvc,
		schedulerSvc:   scheduelrSvc,
	}
	app.SetAliasName("")
	return app, nil
}
