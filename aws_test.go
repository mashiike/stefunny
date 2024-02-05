package stefunny_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSFnService_DescribeStateMachine_NotFound(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("ListStateMachines", mock.Anything, mock.Anything).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{},
	}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, "Hello")
	require.ErrorIs(t, err, stefunny.ErrStateMachineDoesNotExist)
}

func TestSFnService_DescribeStateMachine_SuccessFirstFetch(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("ListStateMachines", mock.Anything, mock.Anything).Return(&sfn.ListStateMachinesOutput{
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
	}, nil).Once()
	m.On("DescribeStateMachine", mock.Anything, &sfn.DescribeStateMachineInput{
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
	}, nil).Once()
	m.On("ListTagsForResource", mock.Anything, &sfn.ListTagsForResourceInput{
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
	}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	sm, err := svc.DescribeStateMachine(ctx, "Hello")
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
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("ListStateMachines", mock.Anything, mock.MatchedBy(
		func(input *sfn.ListStateMachinesInput) bool {
			return input.NextToken == nil
		},
	)).Return(&sfn.ListStateMachinesOutput{
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
	}, nil).Once()
	m.On("ListStateMachines", mock.Anything, mock.MatchedBy(
		func(input *sfn.ListStateMachinesInput) bool {
			return input.NextToken != nil && *input.NextToken == "next"
		},
	)).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Once()

	m.On("DescribeStateMachine", mock.Anything, &sfn.DescribeStateMachineInput{
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
	}, nil).Once()
	m.On("ListTagsForResource", mock.Anything, &sfn.ListTagsForResourceInput{
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
	}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	sm, err := svc.DescribeStateMachine(ctx, "Hello")
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
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	expectedErr := errors.New("this is testing")
	m.On("ListStateMachines", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, "Hello")
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DescribeStateMachine_FailedOnDescribeStateMachine(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("ListStateMachines", mock.Anything, mock.Anything).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Once()
	expectedErr := errors.New("this is testing")
	m.On("DescribeStateMachine", mock.Anything, &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, "Hello")
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DescribeStateMachine_FailedOnListTagsForResource(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("ListStateMachines", mock.Anything, mock.Anything).Return(&sfn.ListStateMachinesOutput{
		StateMachines: []sfntypes.StateMachineListItem{
			{
				Name:            aws.String("Hello"),
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:            sfntypes.StateMachineTypeStandard,
			},
		},
	}, nil).Once()
	m.On("DescribeStateMachine", mock.Anything, &sfn.DescribeStateMachineInput{
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
	}, nil).Once()
	expectedErr := errors.New("this is testing")
	m.On("ListTagsForResource", mock.Anything, &sfn.ListTagsForResourceInput{
		ResourceArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}).Return(nil, expectedErr)
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DescribeStateMachine(ctx, "Hello")
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_CreateNewMachine(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m.On("CreateStateMachine", mock.Anything, mock.MatchedBy(func(input *sfn.CreateStateMachineInput) bool {
		tags := make(map[string]string)
		for _, tag := range input.Tags {
			tags[*tag.Key] = *tag.Value
		}
		result := assert.EqualValues(t, map[string]string{
			"ManagedBy":   "stefunny",
			"Environment": "test",
		}, tags)
		stateMachine.Tags = input.Tags
		return assert.EqualValues(t, stateMachine.CreateStateMachineInput, *input) && result
	})).Return(&sfn.CreateStateMachineOutput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, nil).Once()

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	actual, err := svc.DeployStateMachine(ctx, stateMachine)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.DeployStateMachineOutput{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreationDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		UpdateDate:      aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, actual)
}

func TestSFnService_DeployStateMachine_CreateStateMachineFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m.On("CreateStateMachine", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_UpdateStateMachine(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m.On("UpdateStateMachine", mock.Anything, mock.MatchedBy(func(input *sfn.UpdateStateMachineInput) bool {
		return assert.EqualValues(t, &sfn.UpdateStateMachineInput{
			StateMachineArn:      stateMachine.StateMachineArn,
			Definition:           stateMachine.Definition,
			LoggingConfiguration: stateMachine.LoggingConfiguration,
			RoleArn:              stateMachine.RoleArn,
			TracingConfiguration: stateMachine.TracingConfiguration,
		}, input)

	})).Return(&sfn.UpdateStateMachineOutput{
		UpdateDate: aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
	}, nil).Once()

	m.On("TagResource", mock.Anything, mock.MatchedBy(func(input *sfn.TagResourceInput) bool {
		return assert.EqualValues(t, &sfn.TagResourceInput{
			ResourceArn: stateMachine.StateMachineArn,
			Tags:        stateMachine.CreateStateMachineInput.Tags,
		}, input)
	})).Return(&sfn.TagResourceOutput{}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	actual, err := svc.DeployStateMachine(ctx, stateMachine)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.DeployStateMachineOutput{
		StateMachineArn: stateMachine.StateMachineArn,
		CreationDate:    stateMachine.CreationDate,
		UpdateDate:      aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
	}, actual)
}

func TestSFnService_DeployStateMachine_UpdateStateMachineFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m.On("UpdateStateMachine", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeployStateMachine_TagResourceFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m.On("UpdateStateMachine", mock.Anything, mock.Anything).Return(&sfn.UpdateStateMachineOutput{
		UpdateDate: aws.Time(time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)),
	}, nil).Once()

	m.On("TagResource", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	_, err := svc.DeployStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_DeleteStateMachine_Success(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	stateMachineArn := "arn:aws:states:us-east-1:123456789012:stateMachine:Hello"
	m.On("DeleteStateMachine", mock.Anything, &sfn.DeleteStateMachineInput{
		StateMachineArn: aws.String(stateMachineArn),
	}).Return(&sfn.DeleteStateMachineOutput{}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String(stateMachineArn),
	}
	err := svc.DeleteStateMachine(ctx, stateMachine)
	require.NoError(t, err)
}

func TestSFnService_DeleteStateMachine_Deleting(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
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
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	expectedErr := errors.New("this is testing")
	m.On("DeleteStateMachine", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
	}
	err := svc.DeleteStateMachine(ctx, stateMachine)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_StartExecution_Success(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	m.On("StartExecution", mock.Anything, mock.MatchedBy(
		func(input *sfn.StartExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StartExecutionInput{
				StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
				Name:            aws.String("test"),
				Input:           aws.String(`{"key":"value"}`),
				TraceHeader:     aws.String("Hello_test"),
			}, input)
		},
	)).Return(&sfn.StartExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
	}, nil).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
		},
	}
	output, err := svc.StartExecution(ctx, stateMachine, "test", `{"key":"value"}`)
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.StartExecutionOutput{
		ExecutionArn: "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012",
		StartDate:    time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
	}, output)
}

func TestSFnService_StartExecution_Failed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	expectedErr := errors.New("this is testing")
	m.On("StartExecution", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx := context.Background()
	stateMachine := &stefunny.StateMachine{
		StateMachineArn: aws.String("arn:aws:states:us-east-1:123456789012:stateMachine:Hello"),
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name: aws.String("Hello"),
		},
	}
	_, err := svc.StartExecution(ctx, stateMachine, "test", `{"key":"value"}`)
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_WaitExecution_Success(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	m.On("DescribeExecution", mock.Anything, &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
	}).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusSucceeded,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Once()
	m.On("GetExecutionHistory", mock.Anything, mock.MatchedBy(
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
				Id:        1,
				Timestamp: aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:      sfntypes.HistoryEventTypeExecutionStarted,
			},
			{
				Id:        2,
				Timestamp: aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
				Type:      sfntypes.HistoryEventTypeExecutionSucceeded,
			},
		},
	}, nil).Once()

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	output, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.WaitExecutionOutput{
		Success:   true,
		StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		StopDate:  time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC),
		Output:    `{"key":"value"}`,
	}, output)
}

func TestSFnService_WaitExecution_ExecutionIsFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	m.On("DescribeExecution", mock.Anything, &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
	}).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusFailed,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Once()
	m.On("GetExecutionHistory", mock.Anything, mock.MatchedBy(
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
				Id:        1,
				Timestamp: aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
				Type:      sfntypes.HistoryEventTypeExecutionStarted,
			},
			{
				Id:        2,
				Timestamp: aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
				Type:      sfntypes.HistoryEventTypeExecutionFailed,
				ExecutionFailedEventDetails: &sfntypes.ExecutionFailedEventDetails{
					Error: aws.String("TestError"),
					Cause: aws.String("TestCause"),
				},
			},
		},
	}, nil).Once()

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	output, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.NoError(t, err)
	require.EqualValues(t, &stefunny.WaitExecutionOutput{
		Success:   false,
		Failed:    true,
		StartDate: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
		StopDate:  time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC),
		Output:    `{"key":"value"}`,
		Datail: &sfntypes.ExecutionFailedEventDetails{
			Error: aws.String("TestError"),
			Cause: aws.String("TestCause"),
		},
	}, output)
}

func TestSFnService_WaitExecution_DescribeExecutionAPIFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	expectedErr := errors.New("this is testing")
	m.On("DescribeExecution", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService__WaitExectuion_GetHistoryAPIFailed(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)
	m.On("DescribeExecution", mock.Anything, mock.Anything).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusSucceeded,
		StopDate:     aws.Time(time.Date(2021, 1, 1, 0, 0, 1, 0, time.UTC)),
		Output:       aws.String(`{"key":"value"}`),
	}, nil).Once()
	expectedErr := errors.New("this is testing")
	m.On("GetExecutionHistory", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()
	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.ErrorIs(t, err, expectedErr)
}

func TestSFnService_WaitExecution_ContextCancelStopExection(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("DescribeExecution", mock.Anything, mock.Anything).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusRunning,
	}, nil).Times(2)
	m.On("StopExecution", mock.Anything, mock.MatchedBy(
		func(input *sfn.StopExecutionInput) bool {
			return assert.EqualValues(t, &sfn.StopExecutionInput{
				ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
				Error:        aws.String("stefunny.ContextCanceled"),
				Cause:        aws.String("context canceled"),
			}, input)
		},
	)).Return(&sfn.StopExecutionOutput{}, nil).Once()

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.ErrorIs(t, err, context.Canceled)
}

func TestSFnService_WaitExecution_StopExecutionAPIFaild(t *testing.T) {
	m := NewMockSFnClient(t)
	defer m.AssertExpectations(t)

	m.On("DescribeExecution", mock.Anything, mock.Anything).Return(&sfn.DescribeExecutionOutput{
		ExecutionArn: aws.String("arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012"),
		Name:         aws.String("test"),
		StartDate:    aws.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
		Status:       sfntypes.ExecutionStatusRunning,
	}, nil).Times(2)
	expectedErr := errors.New("this is testing")
	m.On("StopExecution", mock.Anything, mock.Anything).Return(nil, expectedErr).Once()

	svc := stefunny.NewSFnService(m)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.WaitExecution(ctx, "arn:aws:states:us-east-1:123456789012:execution:Hello:12345678-1234-1234-1234-123456789012")
	require.ErrorIs(t, err, context.Canceled)
}
