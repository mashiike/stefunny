package stefunny

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type EventBridgeRule struct {
	eventbridge.PutRuleInput
	RuleArn           *string                   `yaml:"RuleArn,omitempty" json:"RuleArn,omitempty"`
	CreatedBy         *string                   `yaml:"CreatedBy,omitempty" json:"CreatedBy,omitempty"`
	Target            eventbridgetypes.Target   `yaml:"Target,omitempty" json:"Target,omitempty"`
	AdditionalTargets []eventbridgetypes.Target `yaml:"AdditionalTargets,omitempty" json:"AdditionalTargets,omitempty"`
	ConfigFilePath    *string                   `yaml:"ConfigFilePath,omitempty" json:"ConfigFilePath,omitempty"`
}

func (rule *EventBridgeRule) SetStateMachineQualifiedArn(stateMachineArn string) {
	rule.Target.Arn = aws.String(stateMachineArn)
	if rule.Target.Id == nil {
		rule.Target.Id = aws.String(fmt.Sprintf("%s-managed-state-machine", appName))
	}
}

func (rule *EventBridgeRule) IsManagedBy() bool {
	for _, tag := range rule.Tags {
		if coalesce(tag.Key) == tagManagedBy && coalesce(tag.Value) == appName {
			return true
		}
	}
	return false
}

func (rule *EventBridgeRule) AppendTags(tags map[string]string) {
	notExists := make([]eventbridgetypes.Tag, 0, len(tags))
	aleradyExists := make(map[string]string, len(rule.Tags))
	pos := make(map[string]int, len(rule.Tags))
	for i, tag := range rule.Tags {
		aleradyExists[coalesce(tag.Key)] = coalesce(tag.Value)
		pos[coalesce(tag.Key)] = i
	}
	for key, value := range tags {
		if _, ok := aleradyExists[key]; !ok {
			notExists = append(notExists, eventbridgetypes.Tag{
				Key:   aws.String(key),
				Value: aws.String(value),
			})
			continue
		}
		rule.Tags[pos[key]].Value = aws.String(value)
	}
	rule.Tags = append(rule.Tags, notExists...)
}

func (rule *EventBridgeRule) DeleteTag(key string) {
	for i, tag := range rule.Tags {
		if coalesce(tag.Key) == key {
			rule.Tags = append(rule.Tags[:i], rule.Tags[i+1:]...)
			return
		}
	}
}

func (rule *EventBridgeRule) configureJSON() string {
	if rule == nil {
		return "null"
	}
	tags := make(map[string]string, len(rule.Tags))
	for _, tag := range rule.Tags {
		tags[coalesce(tag.Key)] = coalesce(tag.Value)
	}
	return MarshalJSONString(rule.PutRuleInput, map[string]interface{}{
		"Target":            rule.Target,
		"AdditionalTargets": rule.AdditionalTargets,
		"Tags":              tags,
	})
}

func (rule *EventBridgeRule) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString(rule.configureJSON()))
	return builder.String()
}

func (rule *EventBridgeRule) DiffString(newRule *EventBridgeRule, unified bool) string {
	var builder strings.Builder
	from := "[known after apply]"
	if rule != nil {
		from = coalesce(rule.RuleArn, rule.ConfigFilePath, rule.Name)
	}
	to := "[known after apply]"
	if newRule != nil {
		to = coalesce(newRule.RuleArn, newRule.ConfigFilePath, newRule.Name)
	}
	builder.WriteString(
		JSONDiffString(
			rule.configureJSON(), newRule.configureJSON(),
			JSONDiffFromURI(from),
			JSONDiffToURI(to),
			JSONDiffUnified(unified),
		),
	)
	return builder.String()
}

func (rule *EventBridgeRule) SetEnabled(enabled bool) {
	if enabled {
		rule.State = eventbridgetypes.RuleStateEnabled
	} else {
		rule.State = eventbridgetypes.RuleStateDisabled
	}
}

type EventBridgeRules []*EventBridgeRule

func (rules EventBridgeRules) SetStateMachineQualifiedArn(stateMachineArn string) {
	for _, rule := range rules {
		rule.SetStateMachineQualifiedArn(stateMachineArn)
	}
}

func (rules EventBridgeRules) String() string {
	var builder strings.Builder
	for _, rule := range rules {
		builder.WriteString(rule.String())
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (rules EventBridgeRules) SetEnabled(enabled bool) {
	for _, rule := range rules {
		rule.SetEnabled(enabled)
	}
}

func (rules EventBridgeRules) SyncState(other EventBridgeRules) {
	otherMap := make(map[string]*EventBridgeRule, len(other))

	for _, r := range other {
		name := coalesce(r.Name)
		otherMap[name] = r
	}
	for _, r := range rules {
		name := coalesce(r.Name)
		if o, ok := otherMap[name]; ok {
			r.State = o.State
		}
	}
}

func (rules EventBridgeRules) DiffString(newRules EventBridgeRules, unified bool) string {
	result := sliceDiff(rules, newRules, func(r *EventBridgeRule) string {
		return coalesce(r.Name)
	})
	var builder strings.Builder
	var zero *EventBridgeRule
	for _, delete := range result.Delete {
		builder.WriteString(delete.DiffString(zero, unified))
		builder.WriteRune('\n')
	}
	for _, c := range result.Change {
		builder.WriteString(c.Before.DiffString(c.After, unified))
		builder.WriteRune('\n')
	}
	for _, add := range result.Add {
		builder.WriteString(zero.DiffString(add, unified))
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (rules EventBridgeRules) Names() []string {
	names := make([]string, 0, len(rules))
	for _, rule := range rules {
		if name := coalesce(rule.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// sort.Interfaces
func (rules EventBridgeRules) Len() int {
	return len(rules)
}

func (rules EventBridgeRules) Less(i, j int) bool {
	return coalesce(rules[i].Name) < coalesce(rules[j].Name)
}

func (rules EventBridgeRules) Swap(i, j int) {
	rules[i], rules[j] = rules[j], rules[i]
}
