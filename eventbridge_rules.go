package stefunny

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
)

type ScheduleRule struct {
	eventbridge.PutRuleInput
	TargetRoleArn string
	Targets       []eventbridgetypes.Target
}

type ScheduleRules []*ScheduleRule

func (rule *ScheduleRule) SetStateMachineArn(stateMachineArn string) {
	if rule.Description == nil {
		rule.Description = aws.String(fmt.Sprintf("for state machine %s schedule", stateMachineArn))
	}
	if len(rule.Targets) == 0 {
		rule.Targets = []eventbridgetypes.Target{
			{
				RoleArn: &rule.TargetRoleArn,
			},
		}
	}
	rule.Targets[0].Arn = aws.String(stateMachineArn)
	if rule.Targets[0].Id == nil {
		rule.Targets[0].Id = aws.String(fmt.Sprintf("%s-managed-state-machine", appName))
	}
}

func (rule *ScheduleRule) IsManagedBy() bool {
	for _, tag := range rule.Tags {
		if *tag.Key == tagManagedBy && *tag.Value == appName {
			return true
		}
	}
	return false
}

func (rule *ScheduleRule) AppendTags(tags map[string]string) {
	notExists := make([]eventbridgetypes.Tag, 0, len(tags))
	aleradyExists := make(map[string]string, len(rule.Tags))
	pos := make(map[string]int, len(rule.Tags))
	for i, tag := range rule.Tags {
		aleradyExists[*tag.Key] = *tag.Value
		pos[*tag.Key] = i
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

func (rule *ScheduleRule) configureJSON() string {
	tags := make(map[string]string, len(rule.Tags))
	for _, tag := range rule.Tags {
		tags[*tag.Key] = *tag.Value
	}
	params := map[string]interface{}{
		"Name":               rule.Name,
		"Description":        rule.Description,
		"ScheduleExpression": rule.ScheduleExpression,
		"State":              rule.State,
		"Targets":            rule.Targets,
		"Tags":               tags,
	}
	return MarshalJSONString(params)
}

func (rule *ScheduleRule) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString(rule.configureJSON()))
	return builder.String()
}

func (rule *ScheduleRule) DiffString(newRule *ScheduleRule) string {
	var builder strings.Builder
	builder.WriteString(colorRestString(JSONDiffString(rule.configureJSON(), newRule.configureJSON())))
	return builder.String()
}

func (rule *ScheduleRule) SetEnabled(enabled bool) {
	if enabled {
		rule.State = eventbridgetypes.RuleStateEnabled
	} else {
		rule.State = eventbridgetypes.RuleStateDisabled
	}
}

func (rules ScheduleRules) SetStateMachineArn(stateMachineArn string) {
	for _, rule := range rules {
		rule.SetStateMachineArn(stateMachineArn)
	}
}

func (rules ScheduleRules) String() string {
	var builder strings.Builder

	for _, rule := range rules {
		builder.WriteString(rule.String())
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (rules ScheduleRules) SetEnabled(enabled bool) {
	for _, rule := range rules {
		rule.SetEnabled(enabled)
	}
}

func (rules ScheduleRules) SyncState(other ScheduleRules) {
	otherMap := make(map[string]*ScheduleRule, len(other))

	for _, r := range other {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		otherMap[name] = r
	}
	for _, r := range rules {
		name := ""
		if r.Name != nil {
			name = *r.Name
		}
		if o, ok := otherMap[name]; ok {
			r.State = o.State
		}
	}
}

// Things that are in rules but not in other
func (rules ScheduleRules) Subtract(other ScheduleRules) ScheduleRules {
	nothing := make(ScheduleRules, 0, len(rules))
	otherMap := make(map[string]*ScheduleRule, len(other))
	for _, r := range other {
		otherMap[*r.Name] = r
	}
	for _, r := range rules {
		if _, ok := otherMap[*r.Name]; !ok {
			nothing = append(nothing, r)
		}
	}
	return nothing
}

func (rules ScheduleRules) Exclude(other ScheduleRules) ScheduleRules {
	otherMap := make(map[string]*ScheduleRule, len(other))
	for _, r := range other {
		otherMap[*r.Name] = r
	}

	ret := make(ScheduleRules, 0, len(rules))
	ret = append(ret, rules...)
	for i, r := range ret {
		if _, ok := otherMap[*r.Name]; ok {
			ret = append(ret[:i], ret[i+1:]...)
		}
	}
	return ret
}

func (rules ScheduleRules) DiffString(newRules ScheduleRules) string {
	addRuleName := make([]string, 0)
	deleteRuleName := make([]string, 0)
	changeRuleName := make([]string, 0)
	ruleMap := make(map[string]*ScheduleRule, len(rules))
	newRuleMap := make(map[string]*ScheduleRule, len(newRules))

	for _, r := range newRules {
		newRuleMap[*r.Name] = r
	}
	for _, r := range rules {
		ruleMap[*r.Name] = r
		if _, ok := newRuleMap[*r.Name]; ok {
			changeRuleName = append(changeRuleName, *r.Name)
		} else {
			deleteRuleName = append(deleteRuleName, *r.Name)
		}
	}
	for _, r := range newRules {
		if _, ok := ruleMap[*r.Name]; !ok {
			addRuleName = append(addRuleName, *r.Name)
		}
	}

	var builder strings.Builder
	for _, name := range deleteRuleName {
		rule := ruleMap[name]
		builder.WriteString(colorRestString(JSONDiffString(rule.configureJSON(), "null")))
	}
	for _, name := range changeRuleName {
		rule := ruleMap[name]
		newRule := newRuleMap[name]
		builder.WriteString(rule.DiffString(newRule))
	}
	for _, name := range addRuleName {
		newRule := newRuleMap[name]
		builder.WriteString(colorRestString(JSONDiffString("null", newRule.configureJSON())))
	}
	return builder.String()
}
