package stefunny

import (
	"log"
	"strings"
)

// TagStrategy controls how deploy reconciles a state machine's tags with
// the config, and how diff reports tag differences accordingly.
type TagStrategy string

const (
	// TagStrategyAppendOnly adds or updates tags found in the config but
	// never removes a tag that only exists on the live resource.
	TagStrategyAppendOnly TagStrategy = "append_only"
	// TagStrategySync makes the live resource's tags match the config
	// exactly, removing any tag not present in the config.
	TagStrategySync TagStrategy = "sync"
	// TagStrategyNone leaves the live resource's tags untouched.
	TagStrategyNone TagStrategy = "none"
)

// resolveTagStrategy returns TagStrategyAppendOnly when raw is nil (the
// flag was not given), logging a warning since the default is planned to
// change to TagStrategySync in a future release. A non-nil raw is returned
// as-is; its value is validated by the CLI layer's enum tag.
func resolveTagStrategy(raw *string) TagStrategy {
	if raw == nil {
		log.Println("[warn] --tag-strategy not specified, defaulting to append_only (this default is planned to change to sync in a future release)")
		return TagStrategyAppendOnly
	}
	return TagStrategy(*raw)
}

// isAWSReservedTagKey reports whether key uses the "aws:" prefix AWS
// reserves for its own use. Such tags cannot be edited or removed via
// TagResource/UntagResource; see
// https://docs.aws.amazon.com/step-functions/latest/dg/service-quotas.html#sfn-limits-tagging.
func isAWSReservedTagKey(key string) bool {
	return strings.HasPrefix(key, "aws:")
}
