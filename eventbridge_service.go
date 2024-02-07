package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/mashiike/stefunny/internal/eventbridgex"
)

type EventBridgeClient interface {
	eventbridgex.ListRuleNamesByTargetAPIClient
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	DescribeRule(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
	RemoveTargets(ctx context.Context, params *eventbridge.RemoveTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.RemoveTargetsOutput, error)
	ListTagsForResource(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error)
	TagResource(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error)
}

var (
	ErrScheduleRuleDoesNotExist = errors.New("schedule rule does not exist")
	ErrRuleIsNotSchedule        = errors.New("this rule is not schedule")
)

type EventBridgeService interface {
	DescribeScheduleRule(ctx context.Context, ruleName string, optFns ...func(*eventbridge.Options)) (*ScheduleRule, error)
	SearchScheduleRule(ctx context.Context, stateMachineArn string) (ScheduleRules, error)
	DeployScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) (DeployScheduleRulesOutput, error)
	DeleteScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) error
}

var _ EventBridgeService = (*EventBridgeServiceImpl)(nil)

type EventBridgeServiceImpl struct {
	client EventBridgeClient
}

func NewEventBridgeService(client EventBridgeClient) *EventBridgeServiceImpl {
	return &EventBridgeServiceImpl{
		client: client,
	}
}

func (svc *EventBridgeServiceImpl) DescribeScheduleRule(ctx context.Context, ruleName string, optFns ...func(*eventbridge.Options)) (*ScheduleRule, error) {
	describeOutput, err := svc.client.DescribeRule(ctx, &eventbridge.DescribeRuleInput{Name: &ruleName}, optFns...)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil, ErrScheduleRuleDoesNotExist
		}
		return nil, err
	}
	log.Println("[debug] describe rule:", MarshalJSONString(describeOutput))
	if describeOutput.ScheduleExpression == nil {
		return nil, ErrRuleIsNotSchedule
	}
	listTargetsOutput, err := svc.client.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
		Rule:  &ruleName,
		Limit: aws.Int32(5),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] list targets by rule:", MarshalJSONString(listTargetsOutput))
	tagsOutput, err := svc.client.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
		ResourceARN: describeOutput.Arn,
	}, optFns...)
	if err != nil {
		return nil, err
	}
	rule := &ScheduleRule{
		PutRuleInput: eventbridge.PutRuleInput{
			Name:               describeOutput.Name,
			Description:        describeOutput.Description,
			EventBusName:       describeOutput.EventBusName,
			EventPattern:       describeOutput.EventPattern,
			RoleArn:            describeOutput.RoleArn,
			ScheduleExpression: describeOutput.ScheduleExpression,
			State:              describeOutput.State,
			Tags:               tagsOutput.Tags,
		},
		Targets: listTargetsOutput.Targets,
	}
	return rule, nil
}

func (svc *EventBridgeServiceImpl) SearchScheduleRule(ctx context.Context, stateMachineArn string) (ScheduleRules, error) {
	log.Printf("[debug] call SearchScheduleRule(ctx,%s)", stateMachineArn)
	p := eventbridgex.NewListRuleNamesByTargetPaginator(svc.client, &eventbridge.ListRuleNamesByTargetInput{
		TargetArn: aws.String(stateMachineArn),
	})
	rules := make([]*ScheduleRule, 0)
	for p.HasMorePages() {
		output, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, name := range output.RuleNames {
			log.Println("[debug] detect rule: ", name)
			schedule, err := svc.DescribeScheduleRule(ctx, name)
			if err != nil && err != ErrRuleIsNotSchedule {
				return nil, err
			}
			if err == ErrRuleIsNotSchedule {
				continue
			}
			if schedule.IsManagedBy() {
				rules = append(rules, schedule)
			} else {
				name := ""
				if schedule.Name != nil {
					name = *schedule.Name
				}
				log.Printf("[debug] found a scheduled rule `%s` that %s does not manage.", name, appName)
			}
		}
	}
	log.Printf("[debug] end SearchScheduleRule() %d rules found", len(rules))
	return rules, nil
}

type DeployScheduleRuleOutput struct {
	RuleArn          *string
	FailedEntries    []eventbridgetypes.PutTargetsResultEntry
	FailedEntryCount int32
}

func (svc *EventBridgeServiceImpl) DeployScheduleRule(ctx context.Context, rule *ScheduleRule, optFns ...func(*eventbridge.Options)) (*DeployScheduleRuleOutput, error) {
	log.Println("[debug] deploy put rule")
	putRuleOutput, err := svc.client.PutRule(ctx, &rule.PutRuleInput, optFns...)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] deploy put targets")
	putTargetsOutput, err := svc.client.PutTargets(ctx, &eventbridge.PutTargetsInput{
		Rule:    rule.Name,
		Targets: rule.Targets,
	}, optFns...)
	if err != nil {
		return nil, err
	}

	log.Println("[debug] deploy update tag")
	rule.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	_, err = svc.client.TagResource(ctx, &eventbridge.TagResourceInput{
		ResourceARN: putRuleOutput.RuleArn,
		Tags:        rule.PutRuleInput.Tags,
	})
	if err != nil {
		return nil, err
	}
	output := &DeployScheduleRuleOutput{
		RuleArn:          putRuleOutput.RuleArn,
		FailedEntries:    putTargetsOutput.FailedEntries,
		FailedEntryCount: putTargetsOutput.FailedEntryCount,
	}
	return output, nil
}

type DeployScheduleRulesOutput []*DeployScheduleRuleOutput

func (o DeployScheduleRulesOutput) FailedEntryCount() int32 {
	total := int32(0)
	for _, output := range o {
		total += output.FailedEntryCount
	}
	return total
}

func (svc *EventBridgeServiceImpl) DeployScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) (DeployScheduleRulesOutput, error) {
	ret := make([]*DeployScheduleRuleOutput, 0, len(rules))
	for _, rule := range rules {
		output, err := svc.DeployScheduleRule(ctx, rule, optFns...)
		if err != nil {
			return nil, err
		}
		ret = append(ret, output)
	}
	return ret, nil
}

func (svc *EventBridgeServiceImpl) DeleteScheduleRule(ctx context.Context, rule *ScheduleRule, optFns ...func(*eventbridge.Options)) error {
	targetIDs := make([]string, 0, len(rule.Targets))
	for _, target := range rule.Targets {
		targetIDs = append(targetIDs, *target.Id)
	}
	_, err := svc.client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
		Ids:          targetIDs,
		Rule:         rule.Name,
		EventBusName: rule.EventBusName,
	}, optFns...)
	if err != nil {
		return err
	}
	_, err = svc.client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name:         rule.Name,
		EventBusName: rule.EventBusName,
	}, optFns...)
	return err
}

func (svc *EventBridgeServiceImpl) DeleteScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) error {
	for _, rule := range rules {
		if err := svc.DeleteScheduleRule(ctx, rule, optFns...); err != nil {
			return fmt.Errorf("%s :%w", *rule.Name, err)
		}
	}
	return nil
}
