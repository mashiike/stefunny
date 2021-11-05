package stefunny

import (
	"context"
	"fmt"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny/asl"
)

type App struct {
	cfg    *Config
	sfn    *SFnService
	cwlogs *CWLogsService
}

func New(ctx context.Context, cfg *Config) (*App, error) {
	opts := []func(*awsConfig.LoadOptions) error{
		awsConfig.WithRegion(cfg.AWSRegion),
	}
	if endpointsResolver, ok := cfg.EndpointResolver(); ok {
		opts = append(opts, awsConfig.WithEndpointResolver(endpointsResolver))
	}
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	return NewWithClient(cfg, sfn.NewFromConfig(awsCfg), cloudwatchlogs.NewFromConfig(awsCfg))
}

func NewWithClient(cfg *Config, sfnClient SFnClient, logsClient CWLogsClient) (*App, error) {
	return &App{
		cfg:    cfg,
		sfn:    NewSFnService(sfnClient),
		cwlogs: NewCWLogsService(logsClient),
	}, nil
}

func (app *App) Render(ctx context.Context, opt RenderOption) error {
	def, err := app.cfg.LoadDefinition()
	if err != nil {
		return err
	}
	stateMachine, err := asl.Parse(def)
	if err != nil {
		return err
	}
	bs, err := stateMachine.MarshalDOT(app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	_, err = opt.Writer.Write(bs)
	return err
}

func (app *App) LoadLoggingConfiguration(ctx context.Context) (*sfntypes.LoggingConfiguration, error) {
	ret := &sfntypes.LoggingConfiguration{
		Level: sfntypes.LogLevelOff,
	}
	cfg := app.cfg.StateMachine
	if cfg.Logging == nil {
		return ret, nil
	}
	if cfg.Logging.logLevel == sfntypes.LogLevelOff {
		return ret, nil
	}
	ret.IncludeExecutionData = *cfg.Logging.IncludeExecutionData
	arn, err := app.cwlogs.GetLogGroupArn(ctx, cfg.Logging.Destination.LogGroup)
	if err != nil {
		return nil, fmt.Errorf("get log group arn: %w", err)
	}
	ret.Destinations = []sfntypes.LogDestination{
		{
			CloudWatchLogsLogGroup: &sfntypes.CloudWatchLogsLogGroup{
				LogGroupArn: &arn,
			},
		},
	}
	return ret, nil
}
