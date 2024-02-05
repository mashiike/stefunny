package stefunny

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
)

const (
	tagManagedBy = "ManagedBy"
	appName      = "stefunny"
)

type App struct {
	cfg            *Config
	sfnSvc         SFnService
	eventbridgeSvc EventBridgeService
}

type newAppOptions struct {
	mu             sync.Mutex
	cfg            *Config
	sfnSvc         SFnService
	eventbridgeSvc EventBridgeService
	awsCfg         *aws.Config
}

type NewAppOption func(*newAppOptions)

func (o *newAppOptions) GetAWSConfig(ctx context.Context) (aws.Config, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.awsCfg != nil {
		return *o.awsCfg, nil
	}
	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(o.cfg.AWSRegion),
	}
	if endpointsResolver, ok := o.cfg.EndpointResolver(); ok {
		opts = append(opts, awsConfig.WithEndpointResolverWithOptions(endpointsResolver))
	}
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return aws.Config{}, err
	}
	o.awsCfg = &awsCfg
	return awsCfg, nil
}

func (o *newAppOptions) GetSFnService(ctx context.Context) (SFnService, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.sfnSvc != nil {
		return o.sfnSvc, nil
	}
	awsCfg, err := o.GetAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := sfn.NewFromConfig(awsCfg)
	o.sfnSvc = NewSFnService(client)
	return o.sfnSvc, nil
}

func (o *newAppOptions) GetEventBridgeService(ctx context.Context) (EventBridgeService, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.eventbridgeSvc != nil {
		return o.eventbridgeSvc, nil
	}
	awsCfg, err := o.GetAWSConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := eventbridge.NewFromConfig(awsCfg)
	o.eventbridgeSvc = NewEventBridgeService(client)
	return o.eventbridgeSvc, nil
}

// WithSFNClient sets the SFN client for New(ctx, cfg, opts...)
// this is for testing
func WithSFnClient(sfnClient SFnClient) NewAppOption {
	return func(o *newAppOptions) {
		o.sfnSvc = NewSFnService(sfnClient)
	}
}

// WithEventBridgeClient sets the EventBridge client for New(ctx, cfg, opts...)
// this is for testing
func WithEventBridgeClient(eventBridgeClient EventBridgeClient) NewAppOption {
	return func(o *newAppOptions) {
		o.eventbridgeSvc = NewEventBridgeService(eventBridgeClient)
	}
}

// WithAWSConfig sets the AWS config for New(ctx, cfg, opts...)
// this is for testing
func WithAWSConfig(awsCfg aws.Config) NewAppOption {
	return func(o *newAppOptions) {
		o.awsCfg = &awsCfg
	}
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
	app := &App{
		cfg:            cfg,
		sfnSvc:         sfnSvc,
		eventbridgeSvc: eventbridgeSvc,
	}
	return app, nil
}

func (app *App) LoadStateMachine() (*StateMachine, error) {

	stateMachine := &StateMachine{
		CreateStateMachineInput: app.cfg.NewCreateStateMachineInput(),
		Tags:                    app.cfg.Tags,
	}
	stateMachine.Tags[tagManagedBy] = appName
	return stateMachine, nil
}

func (app *App) LoadScheduleRules(_ context.Context, stateMachineArn string) (ScheduleRules, error) {
	rules := make([]*ScheduleRule, 0, len(app.cfg.Schedule))
	for _, cfg := range app.cfg.Schedule {
		rule := &ScheduleRule{
			PutRuleInput: eventbridge.PutRuleInput{
				Name:               aws.String(cfg.RuleName),
				ScheduleExpression: &cfg.Expression,
				State:              eventbridgetypes.RuleStateEnabled,
			},
			Targets: []eventbridgetypes.Target{{
				RoleArn: aws.String(cfg.RoleArn),
			}},
			TargetRoleArn: cfg.RoleArn,
			Tags:          app.cfg.Tags,
		}
		if cfg.Description != "" {
			rule.Description = aws.String(cfg.Description)
		}
		if cfg.ID != "" {
			rule.Targets[0].Id = aws.String(cfg.ID)
		}
		rule.Tags[tagManagedBy] = appName
		rule.SetStateMachineArn(stateMachineArn)
		rules = append(rules, rule)
	}
	return rules, nil
}
