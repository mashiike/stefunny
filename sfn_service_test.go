package stefunny_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSFnService_DescribeStateMachine_NotFound(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{},
	}, nil).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.ErrorIs(t, err, stefunny.ErrStateMachineDoesNotExist)
}

func TestSFnService_DescribeStateMachine_SuccessFirstFetch(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Express"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Express"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeExpress,
			},
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.DescribeStateMachineOutput{
		Name:            aws.String("Hello"),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		Definition:      aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Type:            sfntypes.StateMachineTypeStandard,
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
		Status:          sfntypes.StateMachineStatusActive,
		TracingConfiguration: &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		LoggingConfiguration: &sfntypes.LoggingConfiguration{
			IncludeExecutionData: false,
			Level:                sfntypes.LogLevelOff,
		},
	}, nil).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &sfn.ListTagsForResourceInput{
		ResourceArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.ListTagsForResourceOutput{
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
			{
				Key:   aws.String("Environment"),
				Value: aws.String("test"),
			},
		},
	}, nil).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	sm, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
		Status:          sfntypes.StateMachineStatusActive,
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, sm)
}

func TestSFnService_DescribeStateMachine_SuccessSecondFetch(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Cond(
		func(input *sfn.ListStateMachinesInput) bool {
			return input.NextToken == nil
		},
	), gomock.Any(),
	).Return(&sfn.ListStateMachinesOutput{
		NextToken: aws.String("next"),
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Express"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Express"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeExpress,
			},
			{
				Name:            aws.String("Hoge"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hoge"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Times(1)
	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Cond(
		func(input *sfn.ListStateMachinesInput) bool {
			return input.NextToken != nil && *input.NextToken == "next"
		},
	), gomock.Any()).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Times(1)

	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.DescribeStateMachineOutput{
		Name:            aws.String("Hello"),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		Definition:      aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Type:            sfntypes.StateMachineTypeStandard,
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
		Status:          sfntypes.StateMachineStatusActive,
		TracingConfiguration: &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		LoggingConfiguration: &sfntypes.LoggingConfiguration{
			IncludeExecutionData: false,
			Level:                sfntypes.LogLevelOff,
		},
	}, nil).Times(1)
	m.EXPECT().ListTagsForResource(gomock.Any(), &sfn.ListTagsForResourceInput{
		ResourceArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.ListTagsForResourceOutput{
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("stefunny"),
			},
			{
				Key:   aws.String("Environment"),
				Value: aws.String("test"),
			},
		},
	}, nil).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	sm, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
		Status:          sfntypes.StateMachineStatusActive,
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, sm)
}

func TestSFnService_DescribeStateMachine_FailedOnListStateMachine(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	expectedErr := errors.New("this is testing")

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DescribeStateMachine_FailedOnDescribeStateMachine(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Times(1)
	expectedErr := errors.New("this is testing")
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(nil, expectedErr).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DescribeStateMachine_FailedOnListTagsForResource(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	m.EXPECT().ListStateMachines(gomock.Any(), gomock.Any(), gomock.Any()).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.DescribeStateMachineOutput{
		Name:            aws.String("Hello"),
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		Definition:      aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Type:            sfntypes.StateMachineTypeStandard,
		RoleArn:         aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
		Status:          sfntypes.StateMachineStatusActive,
		TracingConfiguration: &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		LoggingConfiguration: &sfntypes.LoggingConfiguration{
			IncludeExecutionData: false,
			Level:                sfntypes.LogLevelOff,
		},
	}, nil).Times(1)
	expectedErr := errors.New("this is testing")
	m.EXPECT().ListTagsForResource(gomock.Any(), &sfn.ListTagsForResourceInput{
		ResourceArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(nil, expectedErr).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, &stefunny.DescribeStateMachineInput{
		Name: "Hello",
	})
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_CreateNewMachine(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	stateMachine := &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
	}

	m.EXPECT().CreateStateMachine(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, input *sfn.CreateStateMachineInput, opts ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error) {
			tags := make(map[string]string)
			for _, tag := range input.Tags {
				tags[*tag.Key] = *tag.Value
			}
			assert.EqualValues(t, map[string]string{
				"ManagedBy":   "stefunny",
				"Environment": "test",
			}, tags)
			assert.True(t, input.Publish)
			stateMachine.CreateStateMachineInput.Tags = input.Tags
			stateMachine.CreateStateMachineInput.Publish = input.Publish
			return &sfn.CreateStateMachineOutput{
				StateMachineArn:        aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
				CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
			}, nil
		}).Times(1)

	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(&sfn.DescribeStateMachineOutput{
		Name:            stateMachine.CreateStateMachineInput.Name,
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		Definition:      stateMachine.CreateStateMachineInput.Definition,
		Status:          sfntypes.StateMachineStatusActive,
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, nil).Times(1)

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(nil, &sfntypes.ResourceNotFound{
		Message: aws.String("not found"),
	}).Times(1)

	m.EXPECT().CreateStateMachineAlias(gomock.Any(), &sfn.CreateStateMachineAliasInput{
		Name: aws.String("current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
				Weight:                 100,
			},
		},
	}).Return(&sfn.CreateStateMachineAliasOutput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}, nil).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	actual, err := svc.DeployStateMachine(ctx, stateMachine)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.DeployStateMachineOutput{
		StateMachineArn:        aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
		CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdateDate:             aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, actual)
}

func TestSFnService_DeployStateMachine_CreateStateMachineFailed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	stateMachine := &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
	}
	expectedErr := errors.New("this is testing")
	m.EXPECT().CreateStateMachine(gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_UpdateStateMachine(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	stateMachine := &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:          sfntypes.StateMachineStatusActive,
	}

	m.EXPECT().UpdateStateMachine(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, input *sfn.UpdateStateMachineInput, opts ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error) {
			assert.EqualValues(t, &sfn.UpdateStateMachineInput{
				StateMachineArn:      stateMachine.StateMachineArn,
				Definition:           stateMachine.Definition,
				LoggingConfiguration: stateMachine.LoggingConfiguration,
				RoleArn:              stateMachine.RoleArn,
				Publish:              true,
				TracingConfiguration: stateMachine.TracingConfiguration,
			}, input)
			return &sfn.UpdateStateMachineOutput{
				RevisionId:             aws.String("1"),
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
				UpdateDate:             aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
			}, nil
		}).Times(1)

	m.EXPECT().TagResource(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, input *sfn.TagResourceInput, opts ...func(*sfn.Options)) (*sfn.TagResourceOutput, error) {
			assert.EqualValues(t, &sfn.TagResourceInput{
				ResourceArn: stateMachine.StateMachineArn,
				Tags:        stateMachine.CreateStateMachineInput.Tags,
			}, input)
			return &sfn.TagResourceOutput{}, nil
		}).Times(1)

	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: stateMachine.StateMachineArn,
	}).Return(&sfn.DescribeStateMachineOutput{
		Name:            stateMachine.Name,
		StateMachineArn: stateMachine.StateMachineArn,
		Definition:      stateMachine.Definition,
		CreationDate:    stateMachine.CreationDate,
		Status:          sfntypes.StateMachineStatusActive,
	}, nil).Times(1)

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(&sfn.DescribeStateMachineAliasOutput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}, nil).Times(1)

	m.EXPECT().UpdateStateMachineAlias(gomock.Any(), &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
				Weight:                 100,
			},
		},
	}).Return(&sfn.UpdateStateMachineAliasOutput{}, nil).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	actual, err := svc.DeployStateMachine(ctx, stateMachine)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.DeployStateMachineOutput{
		StateMachineArn:        stateMachine.StateMachineArn,
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
		CreationDate:           stateMachine.CreationDate,
		UpdateDate:             aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
	}, actual)
}

func TestSFnService_DeployStateMachine_UpdateStateMachineFailed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	stateMachine := &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:          sfntypes.StateMachineStatusActive,
	}
	expectedErr := errors.New("this is testing")
	m.EXPECT().UpdateStateMachine(gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_TagResourceFailed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()

	stateMachine := &stefunny.StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(`{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`),
			Type:       sfntypes.StateMachineTypeStandard,
			RoleArn:    aws.String("arn:aws:iam::123456789012:role/service-role/StatesExecutionRole-us-east-1"),
			TracingConfiguration: &sfntypes.TracingConfiguration{
				Enabled: false,
			},
			LoggingConfiguration: &sfntypes.LoggingConfiguration{
				IncludeExecutionData: false,
				Level:                sfntypes.LogLevelOff,
			},
			Tags: []sfntypes.Tag{
				{
					Key:   aws.String("ManagedBy"),
					Value: aws.String("stefunny"),
				},
				{
					Key:   aws.String("Environment"),
					Value: aws.String("test"),
				},
			},
		},
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:          sfntypes.StateMachineStatusActive,
	}
	expectedErr := errors.New("this is testing")
	m.EXPECT().UpdateStateMachine(gomock.Any(), gomock.Any()).Return(&sfn.UpdateStateMachineOutput{
		RevisionId:             aws.String("1"),
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
		UpdateDate:             aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
	}, nil).Times(1)

	m.EXPECT().TagResource(gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeleteStateMachine_Success(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachineArn := "arn:aws:states:us-east-1:123456789012:stateMachine:Hello"
	m.EXPECT().DeleteStateMachine(gomock.Any(), &sfn.DeleteStateMachineInput{
		StateMachineArn: aws.String(stateMachineArn),
	}).Return(&sfn.DeleteStateMachineOutput{}, nil).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String(stateMachineArn),
	}
	err := svc.DeleteStateMachine(ctx, stateMachine)
	require.NoError(t, err)
}

func TestSFnService_DeleteStateMachine_Deleting(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		Status:          sfntypes.StateMachineStatusDeleting,
	}
	err := svc.DeleteStateMachine(ctx, stateMachine)
	require.NoError(t, err)
}

func TestSFnService_DeleteStateMachine_Failed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	expectedErr := errors.New("this is testing")
	m.EXPECT().DeleteStateMachine(gomock.Any(), gomock.Any()).Return(nil, expectedErr).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}
	err := svc.DeleteStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_StartExecution_StandardSyncSuccess(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeStandard,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         false,
		Qualifier:     aws.String("current"),
	}
	m.EXPECT().StartExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartExecutionInput{
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(&sfn.StartExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, nil).Times(1)
	m.EXPECT().DescribeExecution(gomock.Any(), &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
	}).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusRunning,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Times(1)
	m.EXPECT().DescribeExecution(gomock.Any(), &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
	}).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusSucceeded,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Times(1)
	m.EXPECT().GetExecutionHistory(gomock.Any(), gomock.Cond(
		func(input *sfn.GetExecutionHistoryInput) bool {
			return assert.EqualValues(t, &sfn.GetExecutionHistoryInput{
				ExecutionArn:         aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
				MaxResults:           5,
				ReverseOrder:         true,
				IncludeExecutionData: aws.Bool(true),
			}, input)
		},
	)).Return(&sfn.GetExecutionHistoryOutput{
		Events: []sfntypes.HistoryEvent{
			{
				Id:                           1,
				Timestamp:                    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:                         sfntypes.HistoryEventTypeExecutionStarted,
				ExecutionStartedEventDetails: &sfntypes.ExecutionStartedEventDetails{},
			},
		},
	}, nil).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	output, err := svc.StartExecution(ctx, stateMachine, params)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StartExecutionOutput{
		ExecutionArn: "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012",
		StartDate:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		Success:      aws.Bool(true),
		Failed:       aws.Bool(false),
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, output)
}

func TestSFnService_StartExecution_StartExecutionFailed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	expectedErr := errors.New("this is testing")
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeStandard,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         false,
	}
	m.EXPECT().StartExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartExecutionInput{
				StateMachineArn: stateMachine.StateMachineArn,
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(nil, expectedErr).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.StartExecution(ctx, stateMachine, params)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_StartExecution_StandardAsyncSuccess(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeStandard,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         true,
	}
	m.EXPECT().StartExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartExecutionInput{
				StateMachineArn: stateMachine.StateMachineArn,
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(&sfn.StartExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, nil).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	output, err := svc.StartExecution(ctx, stateMachine, params)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StartExecutionOutput{
		ExecutionArn: "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012",
		StartDate:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}, output)
}

func TestSFnService_StartExecution_ExpressSyncSuccess(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeExpress,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         false,
	}
	m.EXPECT().StartSyncExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartSyncExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartSyncExecutionInput{
				StateMachineArn: stateMachine.StateMachineArn,
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(&sfn.StartSyncExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.SyncExecutionStatusSucceeded,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Times(1)

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	output, err := svc.StartExecution(ctx, stateMachine, params)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StartExecutionOutput{
		ExecutionArn:      "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012",
		Success:           aws.Bool(true),
		Failed:            aws.Bool(false),
		StartDate:         time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		StopDate:          aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:            aws.String(`{"key":"value"}`),
		CanNotDumpHistory: true,
	}, output)
}

func TestSFnService_StartExecution_ExpressAsyncSuccess(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeExpress,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         true,
	}
	m.EXPECT().StartExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartExecutionInput{
				StateMachineArn: stateMachine.StateMachineArn,
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(
		&sfn.StartExecutionOutput{
			ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
			StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		},
		nil,
	).Times(1)
	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	output, err := svc.StartExecution(ctx, stateMachine, params)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StartExecutionOutput{
		ExecutionArn:      "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012",
		StartDate:         time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		CanNotDumpHistory: true,
	}, output)
}

func TestSFnService_StartExecution_ExpressStartExedcutionFailed(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	expectedErr := errors.New("this is testing")
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
			Type: sfntypes.StateMachineTypeExpress,
		},
	}
	params := &stefunny.StartExecutionInput{
		ExecutionName: "000000-0000-0000-0000-000000000000",
		Input:         "{}",
		Async:         false,
	}
	m.EXPECT().StartSyncExecution(gomock.Any(), gomock.Cond(
		func(input *sfn.StartSyncExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartSyncExecutionInput{
				StateMachineArn: stateMachine.StateMachineArn,
				Name:            aws.String(params.ExecutionName),
				Input:           aws.String(params.Input),
				TraceHeader:     aws.String("Hello_000000-0000-0000-0000-000000000000"),
			}, input)
		},
	)).Return(nil, expectedErr).Times(1)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	svc := stefunny.NewSFnService(m)
	_, err := svc.StartExecution(ctx, stateMachine, params)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService__RollbackStateMachine__NormalCase(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)

	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}
	m.EXPECT().UpdateStateMachineAlias(gomock.Any(), &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
				Weight:                 100,
			},
		},
	}).Return(
		&sfn.UpdateStateMachineAliasOutput{},
		nil,
	).Times(1)
	m.EXPECT().DeleteStateMachineVersion(gomock.Any(), &sfn.DeleteStateMachineVersionInput{
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
	}).Return(
		&sfn.DeleteStateMachineVersionOutput{},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	err := svc.RollbackStateMachine(ctx, stateMachine, false, false)
	require.NoError(t, err)
}

func TestSFnService__RollbackStateMachine__DryRun(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := true
	keepVersion := false
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.NoError(t, err)
}

func TestSFnService__RolebackStateMachine__KeepVersion(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}
	m.EXPECT().UpdateStateMachineAlias(gomock.Any(), &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
				Weight:                 100,
			},
		},
	}).Return(
		&sfn.UpdateStateMachineAliasOutput{},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := false
	keepVersion := true
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.NoError(t, err)
}

func TestSFnService__RollbackStateMachine__DryRunKeepVersion(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := true
	keepVersion := true
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.NoError(t, err)
}

func TestSFnService__RollbackStateMachine__NoVersionToRollback(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)

	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
			CreationDate:    aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("1"),
			Description:     aws.String("test"),
		},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := false
	keepVersion := false
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.ErrorIs(t, err, stefunny.ErrRollbackTargetNotFound)
}

func TestSFnService__RollbackStateMachine__AfterKeepVersion(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}
	m.EXPECT().UpdateStateMachineAlias(gomock.Any(), &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
				Weight:                 100,
			},
		},
	}).Return(
		&sfn.UpdateStateMachineAliasOutput{},
		nil,
	).Times(1)
	m.EXPECT().DeleteStateMachineVersion(gomock.Any(), &sfn.DeleteStateMachineVersionInput{
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
	}).Return(
		&sfn.DeleteStateMachineVersionOutput{},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := false
	keepVersion := false
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.NoError(t, err)
}

func TestSFnService__RollbackStateMachine__OtherVersioinReferenced(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:other"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:other"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:other"),
			Name:                 aws.String("other"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}

	m.EXPECT().UpdateStateMachineAlias(gomock.Any(), &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
				Weight:                 100,
			},
		},
	}).Return(
		&sfn.UpdateStateMachineAliasOutput{},
		nil,
	).Times(1)
	m.EXPECT().DeleteStateMachineVersion(gomock.Any(), &sfn.DeleteStateMachineVersionInput{
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
	}).Return(
		nil,
		&sfntypes.ConflictException{
			Message: aws.String("Version to be deleted must not be referenced by an alias. Current list of aliases referencing this version: [other]"),
		},
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	dryRun := false
	keepVersion := false
	err := svc.RollbackStateMachine(ctx, stateMachine, keepVersion, dryRun)
	require.NoError(t, err)
}

func TestSFnService_PurgeStateMachineVersions_NormalCase(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
			Name:                 aws.String("hoge"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
					CreationDate:           aws.Time(time.Date(2021, 1, 3, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
					CreationDate:           aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	for i := 5; i >= 1; i-- {
		m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
			StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
		}).Return(
			&sfn.DescribeStateMachineOutput{
				StateMachineArn: aws.String(fmt.Sprintf("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:%d", i)),
				CreationDate:    aws.Time(time.Date(2021, 1, i, 0, 0, 0, 0, time.UTC)),
				RevisionId:      aws.String("1"),
				Description:     aws.String("test"),
			},
			nil,
		).Times(1)
	}

	m.EXPECT().DeleteStateMachineVersion(gomock.Any(), &sfn.DeleteStateMachineVersionInput{
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:3"),
	}).Return(
		&sfn.DeleteStateMachineVersionOutput{},
		nil,
	).Times(1)
	m.EXPECT().DeleteStateMachineVersion(gomock.Any(), &sfn.DeleteStateMachineVersionInput{
		StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:1"),
	}).Return(
		&sfn.DeleteStateMachineVersionOutput{},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	err := svc.PurgeStateMachineVersions(ctx, stateMachine, 2)
	require.NoError(t, err)
}

func TestSFnService_PurgeStateMachineVersions_NoVersionToPurge(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:hoge"),
			Name:                 aws.String("hoge"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
					CreationDate:           aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
			CreationDate:    aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("1"),
			Description:     aws.String("test"),
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
			CreationDate:    aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("2"),
			Description:     aws.String("test2"),
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:2"),
			CreationDate:    aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("3"),
			Description:     aws.String("test3"),
		},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	err := svc.PurgeStateMachineVersions(ctx, stateMachine, 2)
	require.NoError(t, err)
}

func TestSFnService_ListStateMachineVersions_Success(t *testing.T) {
	LoggerSetup(t, "debug")
	ctrl := gomock.NewController(t)
	m := mock.NewMockSFnClient(ctrl)
	defer ctrl.Finish()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}

	m.EXPECT().ListStateMachineAliases(gomock.Any(), &sfn.ListStateMachineAliasesInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineAliasesOutput{
			StateMachineAliases: []sfntypes.StateMachineAliasListItem{
				{
					StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachineAlias(gomock.Any(), &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
	}).Return(
		&sfn.DescribeStateMachineAliasOutput{
			StateMachineAliasArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:current"),
			Name:                 aws.String("current"),
			RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					Weight:                 100,
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().ListStateMachineVersions(gomock.Any(), &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	}).Return(
		&sfn.ListStateMachineVersionsOutput{
			StateMachineVersions: []sfntypes.StateMachineVersionListItem{
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
					CreationDate:           aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
				},
				{
					StateMachineVersionArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
					CreationDate:           aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
				},
			},
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5"),
			CreationDate:    aws.Time(time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("1"),
			Description:     aws.String("test"),
		},
		nil,
	).Times(1)
	m.EXPECT().DescribeStateMachine(gomock.Any(), &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
	}).Return(
		&sfn.DescribeStateMachineOutput{
			StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4"),
			CreationDate:    aws.Time(time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC)),
			RevisionId:      aws.String("2"),
			Description:     aws.String("test2"),
		},
		nil,
	).Times(1)

	ctx := context.Background()
	svc := stefunny.NewSFnService(m)
	versions, err := svc.ListStateMachineVersions(ctx, stateMachine)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.ListStateMachineVersionsOutput{
		StateMachineArn: *stateMachine.StateMachineArn,
		Versions: []stefunny.StateMachineVersionListItem{
			{
				StateMachineVersionArn: "arn:aws:states:us-east-1:123456789012:stateMachine:Hello:5",
				Version:                5,
				CreationDate:           time.Date(2021, 1, 5, 0, 0, 0, 0, time.UTC),
				Aliases:                []string{"current"},
				RevisionID:             "1",
				Description:            "test",
			},
			{
				StateMachineVersionArn: "arn:aws:states:us-east-1:123456789012:stateMachine:Hello:4",
				Version:                4,
				CreationDate:           time.Date(2021, 1, 4, 0, 0, 0, 0, time.UTC),
				RevisionID:             "2",
				Description:            "test2",
			},
		},
	}, versions)
}
