package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny/internal/jsonutil"
)

type SFnClient interface {
	sfn.ListStateMachinesAPIClient
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)
	DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	ListTagsForResource(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error)
	TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error)
}

type CWLogsClient interface {
	cloudwatchlogs.DescribeLogGroupsAPIClient
}
type EventBridgeClient interface {
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	ListRuleNamesByTarget(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error)
	DescribeRule(ctx context.Context, params *eventbridge.DescribeRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DescribeRuleOutput, error)
	ListTargetsByRule(ctx context.Context, params *eventbridge.ListTargetsByRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTargetsByRuleOutput, error)
	PutTargets(ctx context.Context, params *eventbridge.PutTargetsInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutTargetsOutput, error)
	DeleteRule(ctx context.Context, params *eventbridge.DeleteRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.DeleteRuleOutput, error)
	ListTagsForResource(ctx context.Context, params *eventbridge.ListTagsForResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListTagsForResourceOutput, error)
	TagResource(ctx context.Context, params *eventbridge.TagResourceInput, optFns ...func(*eventbridge.Options)) (*eventbridge.TagResourceOutput, error)
}

type AWSClients struct {
	SFnClient
	CWLogsClient
	EventBridgeClient
}
type AWSService struct {
	SFnClient
	CWLogsClient
	EventBridgeClient
	cacheArnByGroupName        map[string]string
	cacheStateMachineArnByName map[string]string
}

func NewAWSService(clients AWSClients) *AWSService {
	return &AWSService{
		SFnClient:           clients.SFnClient,
		CWLogsClient:        clients.CWLogsClient,
		EventBridgeClient:   clients.EventBridgeClient,
		cacheArnByGroupName: make(map[string]string),

		cacheStateMachineArnByName: make(map[string]string),
	}
}

var (
	ErrScheduleRuleDoesNotExist = errors.New("schedule rule does not exist")
	ErrRuleIsNotSchedule        = errors.New("this rule is not schedule")
	ErrStateMachineDoesNotExist = errors.New("state machine does not exist")
	ErrLogGroupNotFound         = errors.New("log group not found")
)

type StateMachine struct {
	sfn.CreateStateMachineInput
	CreationDate    *time.Time
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
	Tags            map[string]string
}

func (svc *AWSService) DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*StateMachine, error) {
	arn, err := svc.GetStateMachineArn(ctx, name, optFns...)
	if err != nil {
		return nil, err
	}
	output, err := svc.SFnClient.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: &arn,
	}, optFns...)
	if err != nil {
		if _, ok := err.(*sfntypes.StateMachineDoesNotExist); ok {
			return nil, ErrStateMachineDoesNotExist
		}
		return nil, err
	}
	tagsOutput, err := svc.SFnClient.ListTagsForResource(ctx, &sfn.ListTagsForResourceInput{
		ResourceArn: &arn,
	}, optFns...)
	if err != nil {
		return nil, err
	}
	stateMachine := &StateMachine{
		CreateStateMachineInput: sfn.CreateStateMachineInput{
			Definition:           output.Definition,
			Name:                 output.Name,
			RoleArn:              output.RoleArn,
			LoggingConfiguration: output.LoggingConfiguration,
			TracingConfiguration: output.TracingConfiguration,
			Type:                 output.Type,
			Tags:                 tagsOutput.Tags,
		},
		CreationDate:    output.CreationDate,
		StateMachineArn: output.StateMachineArn,
		Status:          output.Status,
	}
	tags := make(map[string]string, len(tagsOutput.Tags))
	for _, tag := range tagsOutput.Tags {
		tags[*tag.Key] = *tag.Value
	}
	stateMachine.Tags = tags
	return stateMachine, nil
}

func (svc *AWSService) GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error) {
	if arn, ok := svc.cacheStateMachineArnByName[name]; ok {
		return arn, nil
	}
	p := sfn.NewListStateMachinesPaginator(svc.SFnClient, &sfn.ListStateMachinesInput{
		MaxResults: 32,
	})
	for p.HasMorePages() {
		output, err := p.NextPage(ctx, optFns...)
		if err != nil {
			return "", err
		}
		for _, m := range output.StateMachines {
			if *m.Name == name {
				svc.cacheStateMachineArnByName[name] = *m.StateMachineArn
				return svc.cacheStateMachineArnByName[name], nil
			}
		}
	}
	return "", ErrStateMachineDoesNotExist
}

func (svc *AWSService) GetLogGroupArn(ctx context.Context, name string, optFns ...func(*cloudwatchlogs.Options)) (string, error) {
	if arn, ok := svc.cacheArnByGroupName[name]; ok {
		return arn, nil
	}
	p := cloudwatchlogs.NewDescribeLogGroupsPaginator(svc.CWLogsClient, &cloudwatchlogs.DescribeLogGroupsInput{
		Limit:              aws.Int32(32),
		LogGroupNamePrefix: &name,
	})
	for p.HasMorePages() {
		output, err := p.NextPage(ctx, optFns...)
		if err != nil {
			return "", err
		}
		for _, lg := range output.LogGroups {
			if *lg.LogGroupName == name {
				svc.cacheArnByGroupName[name] = *lg.Arn
				return svc.cacheArnByGroupName[name], nil
			}
		}
	}
	return "", ErrLogGroupNotFound
}

type DeployStateMachineOutput struct {
	CreationDate    *time.Time
	UpdateDate      *time.Time
	StateMachineArn *string
}

func (svc *AWSService) DeployStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	var output *DeployStateMachineOutput
	if stateMachine.StateMachineArn == nil {
		log.Println("[debug] try create state machine")
		createOutput, err := svc.SFnClient.CreateStateMachine(ctx, &stateMachine.CreateStateMachineInput, optFns...)
		if err != nil {
			return nil, fmt.Errorf("create failed: %w", err)
		}
		log.Println("[debug] finish create state machine")
		output = &DeployStateMachineOutput{
			StateMachineArn: createOutput.StateMachineArn,
			CreationDate:    createOutput.CreationDate,
			UpdateDate:      createOutput.CreationDate,
		}
	} else {
		var err error
		output, err = svc.updateStateMachine(ctx, stateMachine, optFns...)
		if err != nil {
			return nil, err
		}
	}
	svc.cacheStateMachineArnByName[*stateMachine.Name] = *output.StateMachineArn
	return output, nil
}

func (svc *AWSService) updateStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	log.Println("[debug] try update state machine")
	output, err := svc.SFnClient.UpdateStateMachine(ctx, &sfn.UpdateStateMachineInput{
		StateMachineArn:      stateMachine.StateMachineArn,
		Definition:           stateMachine.Definition,
		LoggingConfiguration: stateMachine.LoggingConfiguration,
		RoleArn:              stateMachine.RoleArn,
		TracingConfiguration: stateMachine.TracingConfiguration,
	}, optFns...)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] finish update state machine")

	log.Println("[debug] try update state machine tags")
	_, err = svc.SFnClient.TagResource(ctx, &sfn.TagResourceInput{
		ResourceArn: stateMachine.StateMachineArn,
		Tags: []sfntypes.Tag{
			{
				Key:   aws.String(tagManagedBy),
				Value: aws.String(appName),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	log.Println("[debug] finish update state machine tags")
	return &DeployStateMachineOutput{
		StateMachineArn: stateMachine.StateMachineArn,
		CreationDate:    stateMachine.CreationDate,
		UpdateDate:      output.UpdateDate,
	}, nil
}

func (svc *AWSService) DeleteStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error {
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s already deleting...\n", *stateMachine.StateMachineArn)
		return nil
	}
	_, err := svc.SFnClient.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
		StateMachineArn: stateMachine.StateMachineArn,
	}, optFns...)
	return err
}

func (s *StateMachine) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(s.configureJSON())
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(*s.Definition)
	return builder.String()
}

func (s *StateMachine) DiffString(newStateMachine *StateMachine) string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(jsonutil.JSONDiffString(s.configureJSON(), newStateMachine.configureJSON()))
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(jsonutil.JSONDiffString(*s.Definition, *newStateMachine.Definition))
	return builder.String()
}

func (s *StateMachine) configureJSON() string {
	params := map[string]interface{}{
		"Name":                 s.Name,
		"RoleArn":              s.RoleArn,
		"LoggingConfiguration": s.LoggingConfiguration,
		"TracingConfiguration": &sfntypes.TracingConfiguration{
			Enabled: false,
		},
		"Type": s.Type,
		"Tags": s.Tags,
	}
	if s.TracingConfiguration != nil {
		params["TracingConfiguration"] = s.TracingConfiguration
	}
	return jsonutil.MarshalJSONString(params)
}

type ScheduleRule struct {
	eventbridge.PutRuleInput
	TargetRoleArn string
	Targets       []eventbridgetypes.Target
	Tags          map[string]string
}

type ScheduleRules []*ScheduleRule

func (svc *AWSService) DescribeScheduleRule(ctx context.Context, ruleName string, optFns ...func(*eventbridge.Options)) (*ScheduleRule, error) {
	describeOutput, err := svc.EventBridgeClient.DescribeRule(ctx, &eventbridge.DescribeRuleInput{Name: &ruleName}, optFns...)
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFoundException") {
			return nil, ErrScheduleRuleDoesNotExist
		}
		return nil, err
	}
	if describeOutput.ScheduleExpression == nil {
		return nil, ErrRuleIsNotSchedule
	}
	listTargetsOutput, err := svc.EventBridgeClient.ListTargetsByRule(ctx, &eventbridge.ListTargetsByRuleInput{
		Rule:  &ruleName,
		Limit: aws.Int32(5),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	tagsOutput, err := svc.EventBridgeClient.ListTagsForResource(ctx, &eventbridge.ListTagsForResourceInput{
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
	tags := make(map[string]string, len(tagsOutput.Tags))
	for _, tag := range tagsOutput.Tags {
		tags[*tag.Key] = *tag.Value
	}
	rule.Tags = tags
	return rule, nil
}

type listRuleNamesByTargetPaginator struct {
	client    EventBridgeClient
	params    *eventbridge.ListRuleNamesByTargetInput
	nextToken *string
	firstPage bool
}

func newListRuleNamesByTargetPaginator(client EventBridgeClient, params *eventbridge.ListRuleNamesByTargetInput) *listRuleNamesByTargetPaginator {
	if params == nil {
		params = &eventbridge.ListRuleNamesByTargetInput{}
	}

	return &listRuleNamesByTargetPaginator{
		client:    client,
		params:    params,
		firstPage: true,
	}
}

func (p *listRuleNamesByTargetPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

func (p *listRuleNamesByTargetPaginator) NextPage(ctx context.Context, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.ListRuleNamesByTarget(ctx, &params, optFns...)
	if err != nil {
		return nil, err
	}
	p.firstPage = false

	prevToken := p.nextToken
	p.nextToken = result.NextToken

	if prevToken != nil && p.nextToken != nil && *prevToken == *p.nextToken {
		p.nextToken = nil
	}
	return result, nil
}

func (svc *AWSService) SearchScheduleRule(ctx context.Context, stateMachineArn string, optFns ...func(*eventbridge.Options)) (ScheduleRules, error) {
	p := newListRuleNamesByTargetPaginator(svc.EventBridgeClient, &eventbridge.ListRuleNamesByTargetInput{
		TargetArn: aws.String(stateMachineArn),
	})
	rules := make([]*ScheduleRule, 0)
	for p.HasMorePages() {
		output, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, name := range output.RuleNames {
			schedule, err := svc.DescribeScheduleRule(ctx, name)
			if err != nil && err != ErrRuleIsNotSchedule {
				return nil, err
			}
			rules = append(rules, schedule)
		}
	}
	return rules, nil
}

type DeployScheduleRuleOutput struct {
	RuleArn          *string
	FailedEntries    []eventbridgetypes.PutTargetsResultEntry
	FailedEntryCount int32
}

func (svc *AWSService) DeployScheduleRule(ctx context.Context, rule *ScheduleRule, optFns ...func(*eventbridge.Options)) (*DeployScheduleRuleOutput, error) {
	log.Println("[debug] deploy put rule")
	putRuleOutput, err := svc.EventBridgeClient.PutRule(ctx, &rule.PutRuleInput, optFns...)
	if err != nil {
		return nil, err
	}
	log.Println("[debug] deploy put targets")
	putTargetsOutput, err := svc.EventBridgeClient.PutTargets(ctx, &eventbridge.PutTargetsInput{
		Rule:    rule.Name,
		Targets: rule.Targets,
	}, optFns...)
	if err != nil {
		return nil, err
	}

	log.Println("[debug] deploy update tag")
	rule.PutRuleInput.Tags = make([]eventbridgetypes.Tag, 0, len(rule.Tags))
	for key, value := range rule.Tags {
		rule.PutRuleInput.Tags = append(rule.PutRuleInput.Tags, eventbridgetypes.Tag{
			Key:   aws.String(key),
			Value: aws.String(value),
		})
	}
	_, err = svc.EventBridgeClient.TagResource(ctx, &eventbridge.TagResourceInput{
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

func (svc *AWSService) DeployScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) (DeployScheduleRulesOutput, error) {
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

func (svc *AWSService) DeleteScheduleRule(ctx context.Context, rule *ScheduleRule, optFns ...func(*eventbridge.Options)) error {
	_, err := svc.EventBridgeClient.DeleteRule(ctx, &eventbridge.DeleteRuleInput{
		Name:         rule.Name,
		EventBusName: rule.EventBusName,
	}, optFns...)
	return err
}

func (svc *AWSService) DeleteScheduleRules(ctx context.Context, rules ScheduleRules, optFns ...func(*eventbridge.Options)) error {
	for _, rule := range rules {
		if err := svc.DeleteScheduleRule(ctx, rule, optFns...); err != nil {
			return fmt.Errorf("%s :%w", *rule.Name, err)
		}
	}
	return nil
}

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

func (rule *ScheduleRule) configureJSON() string {
	params := map[string]interface{}{
		"Name":               rule.Name,
		"Description":        rule.Description,
		"ScheduleExpression": rule.ScheduleExpression,
		"State":              rule.State,
		"Targets":            rule.Targets,
		"Tags":               rule.Tags,
	}
	return jsonutil.MarshalJSONString(params)
}

func (rule *ScheduleRule) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString(rule.configureJSON()))
	return builder.String()
}

func (rule *ScheduleRule) DiffString(newRule *ScheduleRule) string {
	var builder strings.Builder
	builder.WriteString(colorRestString(jsonutil.JSONDiffString(rule.configureJSON(), newRule.configureJSON())))
	return builder.String()
}

func (rule *ScheduleRule) SetEnabled(enabled bool) {
	if enabled {
		rule.State = eventbridgetypes.RuleStateEnabled
	} else {
		rule.State = eventbridgetypes.RuleStateDisabled
	}
}

func (rule *ScheduleRule) HasTagKeyValue(otherKey, otherValue string) bool {
	for key, value := range rule.Tags {
		if key == otherKey && value == otherValue {
			return true
		}
	}
	return false
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
		otherMap[*r.Name] = r
	}
	for _, r := range rules {
		if o, ok := otherMap[*r.Name]; ok {
			r.State = o.State
		}
	}
}

//Things that are in rules but not in other
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
	for i, r := range rules {
		if _, ok := otherMap[*r.Name]; ok {
			rules = append(rules[:i], rules[i+1:]...)
		}
	}
	return rules
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
		builder.WriteString(colorRestString(jsonutil.JSONDiffString(rule.configureJSON(), "null")))
	}
	for _, name := range changeRuleName {
		rule := ruleMap[name]
		newRule := newRuleMap[name]
		builder.WriteString(rule.DiffString(newRule))
	}
	for _, name := range addRuleName {
		newRule := newRuleMap[name]
		builder.WriteString(colorRestString(jsonutil.JSONDiffString("null", newRule.configureJSON())))
	}
	return builder.String()
}
