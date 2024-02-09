package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
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
	ErrEventBridgeRuleDoesNotExist = errors.New("schedule rule does not exist")
)

type EventBridgeService interface {
	SearchRulesByNames(ctx context.Context, ruleNames []string, stateMachineArn string) (EventBridgeRules, error)
	SearchRelatedRules(ctx context.Context, stateMachineArn string) (EventBridgeRules, error)
	DeployRules(ctx context.Context, stateMachineArn string, rules EventBridgeRules, keepState bool) error
}

var _ EventBridgeService = (*EventBridgeServiceImpl)(nil)

type EventBridgeServiceImpl struct {
	client             EventBridgeClient
	cacheRuleByName    map[string]*eventbridge.DescribeRuleOutput
	cacheTargetsByName map[string]*eventbridge.ListTargetsByRuleOutput
	cacheTagsByName    map[string]*eventbridge.ListTagsForResourceOutput
}

func NewEventBridgeService(client EventBridgeClient) *EventBridgeServiceImpl {
	return &EventBridgeServiceImpl{
		client:             client,
		cacheRuleByName:    make(map[string]*eventbridge.DescribeRuleOutput),
		cacheTargetsByName: make(map[string]*eventbridge.ListTargetsByRuleOutput),
		cacheTagsByName:    make(map[string]*eventbridge.ListTagsForResourceOutput),
	}
}

func (svc *EventBridgeServiceImpl) SearchRulesByNames(ctx context.Context, ruleNames []string, stateMachineArn string) (EventBridgeRules, error) {
	log.Printf("[debug] call SearchRulesByNames(ctx,%s)", ruleNames)
	rules := make(EventBridgeRules, 0, len(ruleNames))
	for _, name := range ruleNames {
		rule, err := svc.describeRule(ctx, name, stateMachineArn)
		if err != nil {
			if !errors.Is(err, ErrEventBridgeRuleDoesNotExist) {
				return nil, err
			}
			log.Println("[debug] rule not found", name)
			continue
		}
		log.Println("[debug] rule found", coalesce(rule.Name))
		rules = append(rules, rule)
	}
	sort.Sort(rules)
	log.Printf("[debug] end SearchRulesByNames() %d rules found", len(rules))
	return rules, nil
}

func (svc *EventBridgeServiceImpl) SearchRelatedRules(ctx context.Context, stateMachineArn string) (EventBridgeRules, error) {
	log.Printf("[debug] call SearchRelatedRules(ctx,%s)", stateMachineArn)
	ruleNames, err := svc.searchRelatedRuleNames(ctx, stateMachineArn)
	if err != nil {
		return nil, err
	}
	unqualified := unqualifyARN(stateMachineArn)
	if unqualified != stateMachineArn {
		unqualifiedRelatedRuleNames, err := svc.searchRelatedRuleNames(ctx, unqualified)
		if err != nil {
			return nil, err
		}
		ruleNames = append(ruleNames, unqualifiedRelatedRuleNames...)
		ruleNames = unique(ruleNames)
	}
	rules := make(EventBridgeRules, 0, len(ruleNames))
	for _, name := range ruleNames {
		rule, err := svc.describeRule(ctx, name, stateMachineArn)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	sort.Sort(rules)
	log.Printf("[debug] end SearchRelatedRules() %d rules found", len(rules))
	return rules, nil
}

func (svc *EventBridgeServiceImpl) searchRelatedRuleNames(ctx context.Context, stateMachineArn string) ([]string, error) {
	p := eventbridgex.NewListRuleNamesByTargetPaginator(svc.client, &eventbridge.ListRuleNamesByTargetInput{
		TargetArn: aws.String(stateMachineArn),
	})
	ruleNames := make([]string, 0)
	for p.HasMorePages() {
		output, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		ruleNames = append(ruleNames, output.RuleNames...)
	}
	return ruleNames, nil
}

func (svc *EventBridgeServiceImpl) describeRule(ctx context.Context, ruleName string, stateMachineARN string) (*EventBridgeRule, error) {
	var describeOutput *eventbridge.DescribeRuleOutput
	var ok bool
	var err error
	if describeOutput, ok = svc.cacheRuleByName[ruleName]; !ok {
		describeOutput, err = svc.client.DescribeRule(ctx, &eventbridge.DescribeRuleInput{Name: &ruleName})
		if err != nil {
			if strings.Contains(err.Error(), "ResourceNotFoundException") {
				return nil, ErrEventBridgeRuleDoesNotExist
			}
			return nil, err
		}
		svc.cacheRuleByName[ruleName] = describeOutput
	}
	log.Println("[debug] describe rule:", MarshalJSONString(describeOutput))
	var listTargetsOutput *eventbridge.ListTargetsByRuleOutput
	if listTargetsOutput, ok = svc.cacheTargetsByName[ruleName]; !ok {
		listTargetsOutput, err = svc.client.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
			Rule:  &ruleName,
			Limit: aws.Int32(5),
		})
		if err != nil {
			return nil, err
		}
		svc.cacheTargetsByName[ruleName] = listTargetsOutput
	}
	log.Println("[debug] list targets by rule:", MarshalJSONString(listTargetsOutput))
	var tagsOutput *eventbridge.ListTagsForResourceOutput
	if tagsOutput, ok = svc.cacheTagsByName[*describeOutput.Arn]; !ok {
		tagsOutput, err = svc.client.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
			ResourceARN: describeOutput.Arn,
		})
		if err != nil {
			return nil, err
		}
		svc.cacheTagsByName[*describeOutput.Arn] = tagsOutput
	}
	rule := &EventBridgeRule{
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
		RuleArn: describeOutput.Arn,
	}
	additional := make([]eventbridgetypes.Target, 0, len(listTargetsOutput.Targets))
	var target *eventbridgetypes.Target
	unqualified := unqualifyARN(stateMachineARN)
	log.Printf("[debug] state machine arn: %s", stateMachineARN)
	log.Printf("[debug] unqualified arn: %s", unqualified)
	for i, t := range listTargetsOutput.Targets {
		currentArn := coalesce(t.Arn)
		log.Printf("[debug] current target arn: %s", currentArn)
		if currentArn == stateMachineARN {
			if target != nil {
				additional = append(additional, t)
			}
			log.Println("[debug] found same arn target")
			target = &t
			additional = append(additional, listTargetsOutput.Targets[:i]...)
			break
		}
		if currentArn == unqualified {
			if target != nil {
				additional = append(additional, t)
				continue
			}
			log.Printf("[debug] found unqualified arn target")
			cloned := t
			target = &cloned
			continue
		}
		if unqualifyARN(currentArn) == unqualified {
			if target != nil {
				additional = append(additional, t)
				continue
			}
			log.Printf("[debug] found other alias arn target")
			cloned := t
			target = &cloned
			continue
		}
		additional = append(additional, t)
	}
	rule.AdditionalTargets = additional
	if target == nil {
		return rule, nil
	}
	rule.Target = *target
	return rule, nil
}

func (svc *EventBridgeServiceImpl) DeployRules(ctx context.Context, stateMachineArn string, rules EventBridgeRules, keepState bool) error {
	currentRules, err := svc.SearchRelatedRules(ctx, stateMachineArn)
	if err != nil {
		return err
	}
	if keepState {
		rules.SyncState(currentRules)
	}
	rules.SetStateMachineQualifiedARN(stateMachineArn)
	plan := sliceDiff(currentRules, rules, func(rule *EventBridgeRule) string {
		return coalesce(rule.Name)
	})
	for _, rule := range plan.Delete {
		log.Println("[info] deleting rule:", coalesce(rule.RuleArn))
		if err := svc.deleteRule(ctx, rule); err != nil {
			return fmt.Errorf("delete rule %s: %w", coalesce(rule.Name), err)
		}
	}
	for _, c := range plan.Change {
		log.Println("[info] changing rule:", coalesce(c.Before.RuleArn))
		if err := svc.putRule(ctx, c.After); err != nil {
			return fmt.Errorf("update rule %s: %w", coalesce(c.After.Name), err)
		}
	}
	for _, rule := range plan.Add {
		log.Println("[info] creating rule:", coalesce(rule.Name))
		if err := svc.putRule(ctx, rule); err != nil {
			return fmt.Errorf("create rule %s: %w", coalesce(rule.Name), err)
		}
	}
	return nil
}

func (svc *EventBridgeServiceImpl) putRule(ctx context.Context, rule *EventBridgeRule) error {
	log.Println("[debug] deploy put rule")
	rule.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	putRuleOutput, err := svc.client.PutRule(ctx, &rule.PutRuleInput)
	if err != nil {
		return fmt.Errorf("put rule: %w", err)
	}
	rule.RuleArn = putRuleOutput.RuleArn
	targets := make([]eventbridgetypes.Target, 1, len(rule.AdditionalTargets)+1)
	targets[0] = rule.Target
	targets = append(targets, rule.AdditionalTargets...)
	log.Printf("[debug] deploy put %d targets", len(targets))
	putTargetsOutput, err := svc.client.PutTargets(ctx, &eventbridge.PutTargetsInput{
		Rule:    rule.Name,
		Targets: targets,
	})
	if err != nil {
		return err
	}
	if putTargetsOutput.FailedEntryCount != 0 {
		for _, failed := range putTargetsOutput.FailedEntries {
			log.Printf("[warn] failed to put target: %s", MarshalJSONString(failed))
		}
		return fmt.Errorf("failed to put %d targets", putTargetsOutput.FailedEntryCount)
	}
	log.Println("[debug] deploy update tag")
	rule.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	_, err = svc.client.TagResource(ctx, &eventbridge.TagResourceInput{
		ResourceARN: putRuleOutput.RuleArn,
		Tags:        rule.Tags,
	})
	if err != nil {
		return err
	}
	return nil
}

func (svc *EventBridgeServiceImpl) deleteRule(ctx context.Context, rule *EventBridgeRule) error {
	if !rule.IsManagedBy() {
		log.Printf("[warn] event bridge rule `%s` that %s does not manage. skip delete this rule", coalesce(rule.Name), appName)
		return nil
	}
	targetIDs := make([]string, 0, len(rule.AdditionalTargets)+1)
	targetIDs = append(targetIDs, coalesce(rule.Target.Id))
	for _, target := range rule.AdditionalTargets {
		targetIDs = append(targetIDs, coalesce(target.Id))
	}
	log.Println("[debug] deploy remove targets for rule:", coalesce(rule.Name))
	_, err := svc.client.RemoveTargets(ctx, &eventbridge.RemoveTargetsInput{
		Ids:          targetIDs,
		Rule:         rule.Name,
		EventBusName: rule.EventBusName,
	})
	if err != nil {
		return fmt.Errorf("remove targets: %w", err)
	}
	log.Println("[debug] deploy delete rule:", coalesce(rule.Name))
	_, err = svc.client.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name:         rule.Name,
		EventBusName: rule.EventBusName,
	})
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	return nil
}
