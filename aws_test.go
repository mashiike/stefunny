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
		Tags: map[string]string{
			"ManagedBy":   "stefunny",
			"Environment": "test",
		},
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
		Tags: map[string]string{
			"ManagedBy":   "stefunny",
			"Environment": "test",
		},
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
