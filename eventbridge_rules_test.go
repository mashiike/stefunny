package stefunny_test

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/mashiike/stefunny"
	"github.com/stretchr/testify/require"
)

func TestEventBridgeRule_DiffString__Ignore(t *testing.T) {
	current := &stefunny.EventBridgeRule{
		PutRuleInput: eventbridge.PutRuleInput{
			Name:        aws.String("Hello"),
			Description: aws.String("before"),
		},
	}
	desired := &stefunny.EventBridgeRule{
		PutRuleInput: eventbridge.PutRuleInput{
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

func TestEventBridgeRules_DiffString__ManagedByTagKeyZeroValue(t *testing.T) {
	current := stefunny.EventBridgeRules{
		{
			PutRuleInput: eventbridge.PutRuleInput{
				Name: aws.String("old-rule"),
				Tags: []eventbridgetypes.Tag{
					{Key: aws.String("ManagedBy"), Value: aws.String("stefunny")},
				},
			},
		},
	}
	desired := stefunny.EventBridgeRules{}

	t.Run("zero value falls back to the default ManagedBy tag key", func(t *testing.T) {
		ds, err := current.DiffString(desired, stefunny.DiffStringOption{Unified: false})
		require.NoError(t, err)
		require.NotEmpty(t, strings.TrimSpace(ds), "a rule tagged ManagedBy=stefunny must still be recognized as managed under the zero-value option")
	})

	t.Run("a custom key does not recognize the default ManagedBy tag", func(t *testing.T) {
		ds, err := current.DiffString(desired, stefunny.DiffStringOption{Unified: false, ManagedByTagKey: "Owner"})
		require.NoError(t, err)
		require.Empty(t, strings.TrimSpace(ds), "a rule tagged only with the default key must not be recognized as managed under a custom key")
	})
}
