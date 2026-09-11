package stefunny

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachine struct {
	sfn.CreateStateMachineInput
	CreationDate    *time.Time
	LastUpdateDate  *time.Time
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
	ConfigFilePath  *string
	DefinitionPath  *string
}

// DiffStringOption configures StateMachine.DiffString.
type DiffStringOption struct {
	Unified bool
	// TagStrategy is used by StateMachine.DiffString to project the tag
	// set diff would compare against what deploy would actually leave in
	// place under the same strategy.
	TagStrategy TagStrategy
}

func (s *StateMachine) Source() string {
	if s == nil {
		return knownAfterDeployArn
	}
	if s.StateMachineArn != nil {
		return *s.StateMachineArn
	}
	if s.ConfigFilePath != nil {
		return fmt.Sprintf("state_machine in %s", *s.ConfigFilePath)
	}
	if s.Name != nil {
		return *s.Name
	}
	return knownAfterDeployArn
}

func (s *StateMachine) DefinitionSource() string {
	if s == nil {
		return knownAfterDeployArn
	}
	if s.StateMachineArn != nil {
		return *s.StateMachineArn
	}
	if s.DefinitionPath != nil {
		return *s.DefinitionPath
	}
	if s.Name != nil {
		return *s.Name
	}
	return knownAfterDeployArn
}

func (s *StateMachine) QualifiedArn(name string) string {
	if s == nil {
		return ""
	}
	unqualified := removeQualifierFromArn(coalesce(s.StateMachineArn))
	return addQualifierToArn(unqualified, name)
}

func (s *StateMachine) AppendTags(tags map[string]string) {
	notExists := make([]sfntypes.Tag, 0, len(tags))
	aleradyExists := make(map[string]string, len(s.Tags))
	pos := make(map[string]int, len(s.Tags))
	for i, tag := range s.Tags {
		key := coalesce(tag.Key)
		aleradyExists[key] = coalesce(tag.Value)
		pos[key] = i
	}
	for key, value := range tags {
		if _, ok := aleradyExists[key]; !ok {
			notExists = append(notExists, sfntypes.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
			continue
		}
		s.Tags[pos[key]].Value = aws.String(value)
	}
	s.Tags = append(s.Tags, notExists...)
}

// projectedTags returns the tag set the live resource is expected to have
// immediately after a deploy under tagStrategy, given its current tags and
// the tags desired by config. When exists is false the state machine does
// not exist yet, so tags come from CreateStateMachineInput.Tags regardless
// of strategy: creation is a separate API call from TagResource/
// UntagResource, and strategy only governs those.
func projectedTags(exists bool, currentTags, desiredTags []sfntypes.Tag, tagStrategy TagStrategy) []sfntypes.Tag {
	if !exists {
		return desiredTags
	}
	switch tagStrategy {
	case TagStrategySync:
		return syncMergeTags(currentTags, desiredTags)
	case TagStrategyNone:
		return currentTags
	default:
		return appendOnlyMergeTags(currentTags, desiredTags)
	}
}

// syncMergeTags returns desiredTags with any AWS-reserved tag (an "aws:"
// prefixed key, see
// https://docs.aws.amazon.com/step-functions/latest/dg/service-quotas.html#sfn-limits-tagging)
// that exists only in currentTags appended, keeping its current value.
// UntagResource cannot remove such a tag, so a sync that dropped it here
// would make diff report a removal deploy can never perform. Neither input
// is mutated.
func syncMergeTags(currentTags, desiredTags []sfntypes.Tag) []sfntypes.Tag {
	desiredKeys := make(map[string]struct{}, len(desiredTags))
	for _, tag := range desiredTags {
		desiredKeys[coalesce(tag.Key)] = struct{}{}
	}
	merged := make([]sfntypes.Tag, len(desiredTags), len(desiredTags)+len(currentTags))
	copy(merged, desiredTags)
	for _, tag := range currentTags {
		key := coalesce(tag.Key)
		if _, ok := desiredKeys[key]; ok {
			continue
		}
		if isAWSReservedTagKey(key) {
			merged = append(merged, tag)
		}
	}
	return merged
}

// appendOnlyMergeTags returns desiredTags with any tag that exists only in
// currentTags appended, keeping its current value. Neither input is
// mutated.
func appendOnlyMergeTags(currentTags, desiredTags []sfntypes.Tag) []sfntypes.Tag {
	desiredKeys := make(map[string]struct{}, len(desiredTags))
	for _, tag := range desiredTags {
		desiredKeys[coalesce(tag.Key)] = struct{}{}
	}
	merged := make([]sfntypes.Tag, len(desiredTags), len(desiredTags)+len(currentTags))
	copy(merged, desiredTags)
	for _, tag := range currentTags {
		if _, ok := desiredKeys[coalesce(tag.Key)]; ok {
			continue
		}
		merged = append(merged, tag)
	}
	return merged
}

func (s *StateMachine) DeleteTag(key string) {
	for i, tag := range s.Tags {
		if coalesce(tag.Key) == key {
			s.Tags = append(s.Tags[:i], s.Tags[i+1:]...)
			return
		}
	}
}

func (s *StateMachine) IsManagedBy() bool {
	for _, tag := range s.Tags {
		if coalesce(tag.Key) == tagManagedBy && coalesce(tag.Value) == appName {
			return true
		}
	}
	return false
}

func (s *StateMachine) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(s.configureJSON())
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(*s.Definition)
	return builder.String()
}

// DiffString renders the diff between s (the current state, possibly nil)
// and newStateMachine (the desired state). The tag portion of the diff
// reflects the tag set opt.TagStrategy would actually leave in place, not
// newStateMachine's raw tags, so it stays consistent with what a deploy
// under the same strategy would do.
func (s *StateMachine) DiffString(newStateMachine *StateMachine, opt DiffStringOption) string {
	var builder strings.Builder
	from := s.Source()
	to := newStateMachine.Source()
	var currentTags []sfntypes.Tag
	if s != nil {
		currentTags = s.Tags
	}
	projected := *newStateMachine
	projected.Tags = projectedTags(s != nil, currentTags, newStateMachine.Tags, opt.TagStrategy)
	builder.WriteString(
		JSONDiffString(
			s.configureJSON(),
			projected.configureJSON(),
			JSONDiffUnified(opt.Unified),
			JSONDiffFromURI(from),
			JSONDiffToURI(to),
		),
	)
	def := "null"
	if s != nil {
		def = coalesce(s.Definition)
	}
	from = s.DefinitionSource()
	to = newStateMachine.DefinitionSource()
	builder.WriteString(
		JSONDiffString(
			def,
			coalesce(newStateMachine.Definition),
			JSONDiffUnified(opt.Unified),
			JSONDiffFromURI(from),
			JSONDiffToURI(to),
		),
	)
	return builder.String()
}

func (s *StateMachine) configureJSON() string {
	if s == nil {
		return "null"
	}
	tags := make(map[string]string, len(s.Tags))
	for _, tag := range s.Tags {
		tags[coalesce(tag.Key)] = coalesce(tag.Value)
	}
	params := map[string]interface{}{
		"Name":                 s.Name,
		"RoleArn":              s.RoleArn,
		"LoggingConfiguration": s.LoggingConfiguration,
		"TracingConfiguration": &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		"Type": s.Type,
		"Tags": tags,
	}
	if s.TracingConfiguration != nil {
		params["TracingConfiguration"] = s.TracingConfiguration
	}
	return MarshalJSONString(params)
}
