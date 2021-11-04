package stefunny

import (
	"context"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/mashiike/stefunny/asl"
)

type App struct {
	region string
	cfg    *Config
	sfn    *SFnService
	cwlogs *CWLogsService
}

func New(ctx context.Context, cfg *Config) (*App, error) {
	awsCfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion(cfg.AWSRegion))
	if err != nil {
		return nil, err
	}

	return &App{
		region: awsCfg.Region,
		cfg:    cfg,
		sfn:    NewSFnService(sfn.NewFromConfig(awsCfg)),
		cwlogs: NewCWLogsService(cloudwatchlogs.NewFromConfig(awsCfg)),
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
