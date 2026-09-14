package stefunny_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestInit_ManagedByTagKeyMigration(t *testing.T) {
	LoggerSetup(t, "debug")
	ctx := context.Background()

	mocks := NewMocks(t)
	defer mocks.Finish()

	mocks.sfn.EXPECT().SetAliasName("current").Return()
	mocks.sfn.EXPECT().SetManagedByTagKey("ManagedBy").Return()
	mocks.eventBridge.EXPECT().SetManagedByTagKey("ManagedBy").Return()
	mocks.sfn.EXPECT().SetManagedByTagKey("Owner").Return()
	mocks.eventBridge.EXPECT().SetManagedByTagKey("Owner").Return()

	mocks.sfn.EXPECT().DescribeStateMachine(gomock.Any(), &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	}).Return(
		&stefunny.StateMachine{
			CreateStateMachineInput: sfn.CreateStateMachineInput{
				Name:       aws.String("Hello"),
				RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
				Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
				Tags: []sfntypes.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("stefunny")},
				},
			},
			StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
		},
		nil,
	).Times(1)
	mocks.eventBridge.EXPECT().SearchRelatedRules(gomock.Any(), gomock.Any()).Return(stefunny.EventBridgeRules{}, nil).Times(1)
	mocks.scheduler.EXPECT().SearchRelatedSchedules(gomock.Any(), gomock.Any()).Return(stefunny.Schedules{}, nil).Times(1)

	app, err := stefunny.New(ctx, stefunny.NewDefaultConfig(),
		stefunny.WithSFnService(mocks.sfn),
		stefunny.WithEventBridgeService(mocks.eventBridge),
		stefunny.WithSchedulerService(mocks.scheduler),
	)
	require.NoError(t, err)
	app.SetManagedByTagKey("Owner")

	dir := t.TempDir()
	configPath := filepath.Join(dir, "stefunny.yaml")
	err = app.Init(ctx, stefunny.InitOption{
		StateMachineName: "Hello",
		ConfigPath:       configPath,
	})
	require.NoError(t, err)

	generated, err := os.ReadFile(configPath)
	require.NoError(t, err)
	require.NotContains(t, string(generated), "ManagedBy")
}
