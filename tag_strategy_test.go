package stefunny

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveTagStrategy(t *testing.T) {
	cases := []struct {
		casename string
		raw      *string
		want     TagStrategy
	}{
		{casename: "nil defaults to append_only", raw: nil, want: TagStrategyAppendOnly},
		{casename: "append_only", raw: ptr("append_only"), want: TagStrategyAppendOnly},
		{casename: "sync", raw: ptr("sync"), want: TagStrategySync},
		{casename: "none", raw: ptr("none"), want: TagStrategyNone},
	}
	for _, c := range cases {
		t.Run(c.casename, func(t *testing.T) {
			require.Equal(t, c.want, resolveTagStrategy(c.raw))
		})
	}
}
