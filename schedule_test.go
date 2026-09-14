package stefunny_test

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestSchedule_DiffString__Ignore(t *testing.T) {
	current := &stefunny.Schedule{
		CreateScheduleInput: scheduler.CreateScheduleInput{
			Name:        aws.String("Hello"),
			Description: aws.String("before"),
		},
	}
	desired := &stefunny.Schedule{
		CreateScheduleInput: scheduler.CreateScheduleInput{
			Name:        aws.String("Hello"),
			Description: aws.String("after"),
		},
	}

	ds, err := current.DiffString(desired, stefunny.DiffStringOption{Unified: false})
	require.NoError(t, err)
	require.NotEmpty(t, strings.TrimSpace(ds))

	ds, err = current.DiffString(desired, stefunny.DiffStringOption{Unified: false, Ignore: ".Description"})
	require.NoError(t, err)
	require.Empty(t, strings.TrimSpace(ds))
}
