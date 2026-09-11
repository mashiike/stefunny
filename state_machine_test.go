package stefunny

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/stretchr/testify/require"
)

func tagsOf(kv ...string) []sfntypes.Tag {
	tags := make([]sfntypes.Tag, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		tags = append(tags, sfntypes.Tag{Key: aws.String(kv[i]), Value: aws.String(kv[i+1])})
	}
	return tags
}

func TestProjectedTags(t *testing.T) {
	current := tagsOf("ManagedBy", "stefunny", "Terraform", "owned")
	desired := tagsOf("ManagedBy", "stefunny", "Env", "prod")

	cases := []struct {
		casename    string
		exists      bool
		tagStrategy TagStrategy
		want        []sfntypes.Tag
	}{
		{
			casename:    "does not exist yet, append_only",
			exists:      false,
			tagStrategy: TagStrategyAppendOnly,
			want:        desired,
		},
		{
			casename:    "does not exist yet, sync",
			exists:      false,
			tagStrategy: TagStrategySync,
			want:        desired,
		},
		{
			casename:    "does not exist yet, none",
			exists:      false,
			tagStrategy: TagStrategyNone,
			want:        desired,
		},
		{
			casename:    "exists, append_only merges live-only tags",
			exists:      true,
			tagStrategy: TagStrategyAppendOnly,
			want:        tagsOf("ManagedBy", "stefunny", "Env", "prod", "Terraform", "owned"),
		},
		{
			casename:    "exists, sync replaces with desired",
			exists:      true,
			tagStrategy: TagStrategySync,
			want:        desired,
		},
		{
			casename:    "exists, none leaves current untouched",
			exists:      true,
			tagStrategy: TagStrategyNone,
			want:        current,
		},
	}

	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			got := projectedTags(c.exists, current, desired, c.tagStrategy)
			require.Equal(t, c.want, got)
		})
	}
}

func TestProjectedTags_SyncKeepsAWSReservedTag(t *testing.T) {
	current := tagsOf("ManagedBy", "stefunny", "aws:cloudformation:stack-name", "my-stack")
	desired := tagsOf("ManagedBy", "stefunny", "Env", "prod")

	got := projectedTags(true, current, desired, TagStrategySync)
	require.ElementsMatch(t, tagsOf("ManagedBy", "stefunny", "Env", "prod", "aws:cloudformation:stack-name", "my-stack"), got)
}

func TestAppendOnlyMergeTags(t *testing.T) {
	current := tagsOf("ManagedBy", "stefunny", "Terraform", "owned")
	desired := tagsOf("ManagedBy", "stefunny", "Env", "prod")

	got := appendOnlyMergeTags(current, desired)
	require.ElementsMatch(t, tagsOf("ManagedBy", "stefunny", "Env", "prod", "Terraform", "owned"), got)

	require.Equal(t, tagsOf("ManagedBy", "stefunny", "Terraform", "owned"), current, "currentTags must not be mutated")
	require.Equal(t, tagsOf("ManagedBy", "stefunny", "Env", "prod"), desired, "desiredTags must not be mutated")
}

func TestAppendOnlyMergeTags_DesiredValueWins(t *testing.T) {
	current := tagsOf("Env", "dev")
	desired := tagsOf("Env", "prod")

	got := appendOnlyMergeTags(current, desired)
	require.Equal(t, tagsOf("Env", "prod"), got)
}

// TestStateMachine_DiffString_TagStrategy_ConfigOnlyTag pins down the
// behavior that distinguishes TagStrategyNone from the other strategies for
// diff: a tag declared only in config (not yet on the live resource) is
// still reported as a difference under append_only/sync (deploy would add
// it), but never under none (deploy would never touch tags at all).
func TestStateMachine_DiffString_TagStrategy_ConfigOnlyTag(t *testing.T) {
	definition := `{"StartAt":"Hello","States":{"Hello":{"Type":"Pass","End":true}}}`
	current := &StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(definition),
			Tags:       tagsOf("ManagedBy", "stefunny"),
		},
		StateMachineArn: aws.String("arn:aws:states:us-east-1:000000000000:stateMachine:Hello"),
	}
	desired := &StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Name:       aws.String("Hello"),
			Definition: aws.String(definition),
			Tags:       tagsOf("ManagedBy", "stefunny", "Env", "prod"),
		},
	}

	cases := []struct {
		tagStrategy TagStrategy
		wantDiff    bool
	}{
		{tagStrategy: TagStrategyAppendOnly, wantDiff: true},
		{tagStrategy: TagStrategySync, wantDiff: true},
		{tagStrategy: TagStrategyNone, wantDiff: false},
	}
	for _, c := range cases {
		t.Run(string(c.tagStrategy), func(t *testing.T) {
			ds := strings.TrimSpace(current.DiffString(desired, DiffStringOption{TagStrategy: c.tagStrategy}))
			if c.wantDiff {
				require.NotEmpty(t, ds)
			} else {
				require.Empty(t, ds)
			}
		})
	}
}
