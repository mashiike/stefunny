package stefunny

import (
	"context"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
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
