package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/eventbridge"
	eventbridgetypes "github.com/aws/aws-sdk-go-v2/service/eventbridge/types"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/shogo82148/go-retry"
)

const (
	currentAliasName = "current"
)

type SFnClient interface {
	sfn.ListStateMachinesAPIClient
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	CreateStateMachineAlias(ctx context.Context, params *sfn.CreateStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineAliasOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	DescribeStateMachineAlias(ctx context.Context, params *sfn.DescribeStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineAliasOutput, error)
	ListStateMachineVersions(ctx context.Context, params *sfn.ListStateMachineVersionsInput, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineVersionsOutput, error)
	UpdateStateMachine(ctx context.Context, params *sfn.UpdateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineOutput, error)
	UpdateStateMachineAlias(ctx context.Context, params *sfn.UpdateStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.UpdateStateMachineAliasOutput, error)
	DeleteStateMachine(ctx context.Context, params *sfn.DeleteStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineOutput, error)
	DeleteStateMachineVersion(ctx context.Context, params *sfn.DeleteStateMachineVersionInput, optFns ...func(*sfn.Options)) (*sfn.DeleteStateMachineVersionOutput, error)
	ListTagsForResource(ctx context.Context, params *sfn.ListTagsForResourceInput, optFns ...func(*sfn.Options)) (*sfn.ListTagsForResourceOutput, error)
	StartExecution(ctx context.Context, params *sfn.StartExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartExecutionOutput, error)
	StartSyncExecution(ctx context.Context, params *sfn.StartSyncExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StartSyncExecutionOutput, error)
	DescribeExecution(ctx context.Context, params *sfn.DescribeExecutionInput, optFns ...func(*sfn.Options)) (*sfn.DescribeExecutionOutput, error)
	StopExecution(ctx context.Context, params *sfn.StopExecutionInput, optFns ...func(*sfn.Options)) (*sfn.StopExecutionOutput, error)
	GetExecutionHistory(ctx context.Context, params *sfn.GetExecutionHistoryInput, optFns ...func(*sfn.Options)) (*sfn.GetExecutionHistoryOutput, error)
	TagResource(ctx context.Context, params *sfn.TagResourceInput, optFns ...func(*sfn.Options)) (*sfn.TagResourceOutput, error)
}

type CloudWatchLogsClient interface {
	cloudwatchlogs.DescribeLogGroupsAPIClient
}

type EventBridgeClient interface {
	PutRule(ctx context.Context, params *eventbridge.PutRuleInput, optFns ...func(*eventbridge.Options)) (*eventbridge.PutRuleOutput, error)
	ListRuleNamesByTarget(ctx context.Context, params *eventbridge.ListRuleNamesByTargetInput, optFns ...func(*eventbridge.Options)) (*eventbridge.ListRuleNamesByTargetOutput, error)
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
	ErrStateMachineDoesNotExist = errors.New("state machine does not exist")
)

type SFnService interface {
	DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*StateMachine, error)
	GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error)
	DeployStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error)
	DeleteStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error
	RollbackStateMachine(ctx context.Context, stateMachine *StateMachine, keepVersion bool, dryRun bool, optFns ...func(*sfn.Options)) error
	WaitExecution(ctx context.Context, executionArn string) (*WaitExecutionOutput, error)
	StartExecution(ctx context.Context, stateMachine *StateMachine, executionName, input string) (*StartExecutionOutput, error)
	StartSyncExecution(ctx context.Context, stateMachine *StateMachine, executionName, input string) (*sfn.StartSyncExecutionOutput, error)
	GetExecutionHistory(ctx context.Context, executionArn string) ([]HistoryEvent, error)
}

type SFnServiceImpl struct {
	client                      SFnClient
	cacheStateMachineArnByName  map[string]string
	cacheStateMAchineAliasByArn map[string]*sfn.DescribeStateMachineAliasOutput
	retryPolicy                 retry.Policy
}

var _ SFnService = (*SFnServiceImpl)(nil)

func NewSFnService(client SFnClient) *SFnServiceImpl {
	return &SFnServiceImpl{
		client:                      client,
		cacheStateMachineArnByName:  make(map[string]string),
		cacheStateMAchineAliasByArn: make(map[string]*sfn.DescribeStateMachineAliasOutput),
		retryPolicy: retry.Policy{
			MinDelay: time.Second,
			MaxDelay: 10 * time.Second,
			MaxCount: 30,
		},
	}
}

func (svc *SFnServiceImpl) DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*StateMachine, error) {
	arn, err := svc.GetStateMachineArn(ctx, name, optFns...)
	if err != nil {
		return nil, err
	}
	output, err := svc.client.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: &arn,
	}, optFns...)
	if err != nil {
		if _, ok := err.(*sfntypes.StateMachineDoesNotExist); ok {
			return nil, ErrStateMachineDoesNotExist
		}
		return nil, err
	}
	tagsOutput, err := svc.client.ListTagsForResource(ctx, &sfn.ListTagsForResourceInput{
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
	return stateMachine, nil
}

func (svc *SFnServiceImpl) GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error) {
	if arn, ok := svc.cacheStateMachineArnByName[name]; ok {
		return arn, nil
	}
	p := sfn.NewListStateMachinesPaginator(svc.client, &sfn.ListStateMachinesInput{
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

type DeployStateMachineOutput struct {
	CreationDate           *time.Time
	UpdateDate             *time.Time
	StateMachineArn        *string
	StateMachineVersionArn *string
}

func (svc *SFnServiceImpl) DeployStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	var output *DeployStateMachineOutput
	stateMachine.AppendTags(map[string]string{
		tagManagedBy: appName,
	})
	if stateMachine.StateMachineArn == nil {
		log.Println("[debug] try create state machine")
		cloned := stateMachine.CreateStateMachineInput
		cloned.Publish = true
		createOutput, err := svc.client.CreateStateMachine(ctx, &cloned, optFns...)
		if err != nil {
			return nil, fmt.Errorf("create failed: %w", err)
		}
		log.Printf("[info] create state machine `%s`", *createOutput.StateMachineVersionArn)
		log.Println("[debug] finish create state machine")
		output = &DeployStateMachineOutput{
			StateMachineArn:        createOutput.StateMachineArn,
			StateMachineVersionArn: createOutput.StateMachineVersionArn,
			CreationDate:           createOutput.CreationDate,
			UpdateDate:             createOutput.CreationDate,
		}
		stateMachine.StateMachineArn = createOutput.StateMachineArn
		stateMachine.CreationDate = createOutput.CreationDate
		stateMachine.Status = sfntypes.StateMachineStatusActive
	} else {
		var err error
		output, err = svc.updateStateMachine(ctx, stateMachine, optFns...)
		if err != nil {
			return nil, err
		}
		log.Printf("[info] update state machine `%s`", *output.StateMachineVersionArn)
	}
	svc.cacheStateMachineArnByName[*stateMachine.Name] = *output.StateMachineArn
	if err := svc.waitForLastUpdateStatusActive(ctx, stateMachine, optFns...); err != nil {
		return nil, fmt.Errorf("wait for last update status active failed: %w", err)
	}
	if err := svc.updateCurrentArias(ctx, stateMachine, *output.StateMachineVersionArn, optFns...); err != nil {
		return nil, fmt.Errorf("update current alias failed: %w", err)
	}
	return output, nil
}

func (svc *SFnServiceImpl) updateStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error) {
	log.Println("[debug] try update state machine")
	output, err := svc.client.UpdateStateMachine(ctx, &sfn.UpdateStateMachineInput{
		StateMachineArn:      stateMachine.StateMachineArn,
		Definition:           stateMachine.Definition,
		LoggingConfiguration: stateMachine.LoggingConfiguration,
		RoleArn:              stateMachine.RoleArn,
		TracingConfiguration: stateMachine.TracingConfiguration,
		Publish:              true,
		VersionDescription:   stateMachine.VersionDescription,
	}, optFns...)
	if err != nil {
		return nil, err
	}
	log.Printf("[debug] revision_id = `%s`", *output.RevisionId)
	log.Println("[debug] finish update state machine")

	log.Println("[debug] try update state machine tags")
	_, err = svc.client.TagResource(ctx, &sfn.TagResourceInput{
		ResourceArn: stateMachine.StateMachineArn,
		Tags:        stateMachine.Tags,
	})
	if err != nil {
		return nil, err
	}
	log.Println("[debug] finish update state machine tags")
	return &DeployStateMachineOutput{
		StateMachineArn:        stateMachine.StateMachineArn,
		StateMachineVersionArn: output.StateMachineVersionArn,
		CreationDate:           stateMachine.CreationDate,
		UpdateDate:             output.UpdateDate,
	}, nil
}

func (svc *SFnServiceImpl) describeStateMachineAlias(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineAliasOutput, error) {
	aliasArn := fmt.Sprintf("%s:%s", *stateMachine.StateMachineArn, currentAliasName)
	if alias, ok := svc.cacheStateMAchineAliasByArn[aliasArn]; ok {
		return alias, nil
	}
	alias, err := svc.client.DescribeStateMachineAlias(ctx, &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String(aliasArn),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	svc.cacheStateMAchineAliasByArn[*stateMachine.StateMachineArn] = alias
	return alias, nil
}

func (svc *SFnServiceImpl) updateCurrentArias(ctx context.Context, stateMachine *StateMachine, versionARN string, optFns ...func(*sfn.Options)) error {
	alias, err := svc.describeStateMachineAlias(ctx, stateMachine, optFns...)
	if err != nil {
		var notExists *sfntypes.ResourceNotFound
		if errors.As(err, &notExists) {
			log.Println("[info] current alias does not exist, create it...")
			output, err := svc.client.CreateStateMachineAlias(ctx, &sfn.CreateStateMachineAliasInput{
				Name: aws.String(currentAliasName),
				RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
					{
						StateMachineVersionArn: aws.String(versionARN),
						Weight:                 100,
					},
				},
			}, optFns...)
			if err != nil {
				return err
			}
			log.Printf("[info] create current alias `%s`", *output.StateMachineAliasArn)
			return nil
		}
		return err
	}
	log.Printf("[info] update current alias `%s`", *alias.StateMachineAliasArn)
	_, err = svc.client.UpdateStateMachineAlias(ctx, &sfn.UpdateStateMachineAliasInput{
		StateMachineAliasArn: alias.StateMachineAliasArn,
		RoutingConfiguration: []sfntypes.RoutingConfigurationListItem{
			{
				StateMachineVersionArn: aws.String(versionARN),
				Weight:                 100,
			},
		},
	}, optFns...)
	if err != nil {
		return err
	}
	return nil
}

func (svc *SFnServiceImpl) waitForLastUpdateStatusActive(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error {
	retrier := svc.retryPolicy.Start(ctx)
	for retrier.Continue() {
		output, err := svc.client.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
			StateMachineArn: stateMachine.StateMachineArn,
		}, optFns...)
		if err != nil {
			var apiErr smithy.APIError
			if !errors.As(err, &apiErr) {
				log.Printf("[debug] unexpected error: %s", err)
			}
			if apiErr.ErrorCode() == "AccessDeniedException" {
				log.Println("[debug] access denied, skip wait")
				return err
			}
			log.Println("[warn] describe state machine failed, retrying... :", err)
			continue
		}
		if output.Status == sfntypes.StateMachineStatusActive {
			return nil
		}
		log.Printf(
			"[info] waiting for StateMachine `%s`: current status is `%s`",
			sfntypes.StateMachineStatusActive, output.Status,
		)
	}
	return errors.New("max retry count exceeded")
}

func extructVersion(versionARN string) (int, error) {
	arnObj, err := arn.Parse(versionARN)
	if err != nil {
		return 0, fmt.Errorf("parse arn failed: %w", err)
	}
	parts := strings.Split(arnObj.Resource, ":")
	if parts[0] != "stateMachine" {
		return 0, fmt.Errorf("`%s` is not state machine version arn", versionARN)
	}
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid arn format: %s", versionARN)
	}
	version, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("parse version number failed: %w", err)
	}
	return version, nil
}

func (svc *SFnServiceImpl) RollbackStateMachine(ctx context.Context, stateMachine *StateMachine, keepVersion bool, dryRun bool, optFns ...func(*sfn.Options)) error {
	if stateMachine.StateMachineArn == nil {
		return ErrStateMachineDoesNotExist
	}
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s already deleting...\n", *stateMachine.StateMachineArn)
		return nil
	}
	alias, err := svc.describeStateMachineAlias(ctx, stateMachine, optFns...)
	if err != nil {
		var notExists *sfntypes.ResourceNotFound
		if errors.As(err, &notExists) {
			log.Println("[notice] current alias does not exist, can not rollback")
			return nil
		}
		return err
	}
	if len(alias.RoutingConfiguration) > 1 {
		log.Println("[notice] current alias has multiple versions, can not rollback, please manual rollback")
		return nil
	}
	currentVersionArn := *alias.RoutingConfiguration[0].StateMachineVersionArn
	currentVersion, err := extructVersion(currentVersionArn)
	if err != nil {
		return fmt.Errorf("extruct version failed: %w", err)
	}
	log.Printf("[info] current alias version is `%d`", currentVersion)
	if currentVersion <= 1 {
		log.Println("[notice] current alias has no previous version, can not rollback")
		return nil
	}
	p := newListStateMachineVersionsPaginator(svc.client, &sfn.ListStateMachineVersionsInput{
		StateMachineArn: stateMachine.StateMachineArn,
		MaxResults:      32,
	})
	targetVersion := 0
	targetVersionArn := currentVersionArn
	for p.HasMorePages() {
		output, err := p.NextPage(ctx, optFns...)
		if err != nil {
			return fmt.Errorf("list state machine versions failed: %w", err)
		}
		for _, v := range output.StateMachineVersions {
			log.Println("[debug] found version: ", *v.StateMachineVersionArn)
			version, err := extructVersion(*v.StateMachineVersionArn)
			if err != nil {
				return fmt.Errorf("list state machine: extruct version failed: %w", err)
			}
			log.Printf("[debug] check rollback target selectable: %d > %d = %t", version, currentVersion, version > currentVersion)
			if version >= currentVersion {
				log.Println("[debug] skip version: ", version)
				continue
			}
			log.Printf("[debug] check latest rollback target: %d < %d = %t", targetVersion, version, targetVersion < version)
			if targetVersion < version {
				targetVersion = version
				targetVersionArn = *v.StateMachineVersionArn
			}
		}
	}
	log.Println("[debug] target version: ", targetVersion)
	if targetVersionArn == currentVersionArn {
		log.Println("[notice] no previous version found, can not rollback")
		return nil
	}
	log.Printf("[info] rollback to version `%d`", targetVersion)
	if !dryRun {
		if err := svc.updateCurrentArias(ctx, stateMachine, targetVersionArn, optFns...); err != nil {
			return fmt.Errorf("update current alias failed: %w", err)
		}
		log.Println("[info] rollback success")
	}
	if keepVersion {
		return nil
	}
	log.Printf("[info] delete version `%d`", currentVersion)
	if !dryRun {
		_, err = svc.client.DeleteStateMachineVersion(ctx, &sfn.DeleteStateMachineVersionInput{
			StateMachineVersionArn: &currentVersionArn,
		}, optFns...)
		if err != nil {
			return fmt.Errorf("delete version failed: %w", err)
		}
		log.Printf("[info] `%s` deleted", currentVersionArn)
	}
	return nil
}

func (svc *SFnServiceImpl) DeleteStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error {
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s already deleting...\n", *stateMachine.StateMachineArn)
		return nil
	}
	_, err := svc.client.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
		StateMachineArn: stateMachine.StateMachineArn,
	}, optFns...)
	return err
}

type StateMachine struct {
	sfn.CreateStateMachineInput
	CreationDate    *time.Time
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
}

func (s *StateMachine) AppendTags(tags map[string]string) {
	notExists := make([]sfntypes.Tag, 0, len(tags))
	aleradyExists := make(map[string]string, len(s.Tags))
	pos := make(map[string]int, len(s.Tags))
	for i, tag := range s.Tags {
		aleradyExists[*tag.Key] = *tag.Value
		pos[*tag.Key] = i
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

func (s *StateMachine) IsManagedBy() bool {
	for _, tag := range s.Tags {
		if *tag.Key == tagManagedBy && *tag.Value == appName {
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

func (s *StateMachine) DiffString(newStateMachine *StateMachine) string {
	var builder strings.Builder
	builder.WriteString(colorRestString("StateMachine Configure:\n"))
	builder.WriteString(JSONDiffString(s.configureJSON(), newStateMachine.configureJSON()))
	builder.WriteString(colorRestString("\nStateMachine Definition:\n"))
	builder.WriteString(JSONDiffString(*s.Definition, *newStateMachine.Definition))
	return builder.String()
}

func (s *StateMachine) configureJSON() string {
	tags := make(map[string]string, len(s.Tags))
	for _, tag := range s.Tags {
		tags[*tag.Key] = *tag.Value
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

type ScheduleRule struct {
	eventbridge.PutRuleInput
	TargetRoleArn string
	Targets       []eventbridgetypes.Target
}

type ScheduleRules []*ScheduleRule

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

type ListStateMachineVersionsPaginator struct {
	client    SFnClient
	params    *sfn.ListStateMachineVersionsInput
	nextToken *string
	firstPage bool
}

func newListStateMachineVersionsPaginator(client SFnClient, params *sfn.ListStateMachineVersionsInput) *ListStateMachineVersionsPaginator {
	if params == nil {
		params = &sfn.ListStateMachineVersionsInput{}
	}

	return &ListStateMachineVersionsPaginator{
		client:    client,
		params:    params,
		firstPage: true,
	}
}

func (p *ListStateMachineVersionsPaginator) HasMorePages() bool {
	return p.firstPage || p.nextToken != nil
}

func (p *ListStateMachineVersionsPaginator) NextPage(ctx context.Context, optFns ...func(*sfn.Options)) (*sfn.ListStateMachineVersionsOutput, error) {
	if !p.HasMorePages() {
		return nil, fmt.Errorf("no more pages available")
	}

	params := *p.params
	params.NextToken = p.nextToken

	result, err := p.client.ListStateMachineVersions(ctx, &params, optFns...)
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

func (svc *EventBridgeServiceImpl) SearchScheduleRule(ctx context.Context, stateMachineArn string) (ScheduleRules, error) {
	log.Printf("[debug] call SearchScheduleRule(ctx,%s)", stateMachineArn)
	p := newListRuleNamesByTargetPaginator(svc.client, &eventbridge.ListRuleNamesByTargetInput{
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

type StartExecutionOutput struct {
	ExecutionArn string
	StartDate    time.Time
}

func (svc *SFnServiceImpl) StartExecution(ctx context.Context, stateMachine *StateMachine, executionName, input string) (*StartExecutionOutput, error) {
	if executionName == "" {
		uuidObj, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		executionName = uuidObj.String()
	}
	output, err := svc.client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: stateMachine.StateMachineArn,
		Input:           aws.String(input),
		Name:            aws.String(executionName),
		TraceHeader:     aws.String(*stateMachine.Name + "_" + executionName),
	})
	if err != nil {
		return nil, err
	}
	return &StartExecutionOutput{
		ExecutionArn: *output.ExecutionArn,
		StartDate:    *output.StartDate,
	}, nil
}

type WaitExecutionOutput struct {
	Success   bool
	Failed    bool
	StartDate time.Time
	StopDate  time.Time
	Output    string
	Datail    interface{}
}

func (o *WaitExecutionOutput) Elapsed() time.Duration {
	return o.StopDate.Sub(o.StartDate)
}

func (svc *SFnServiceImpl) WaitExecution(ctx context.Context, executionArn string) (*WaitExecutionOutput, error) {
	input := &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionArn),
	}
	output, err := svc.client.DescribeExecution(ctx, input)
	if err != nil {
		return nil, err
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for output.Status == sfntypes.ExecutionStatusRunning {
		log.Printf("[info] execution status: %s", output.Status)
		select {
		case <-ctx.Done():
			stopCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			log.Printf("[warn] try stop execution: %s", executionArn)
			result := &WaitExecutionOutput{
				Success: false,
				Failed:  false,
			}
			output, err = svc.client.DescribeExecution(stopCtx, input)
			if err != nil {
				return result, err
			}
			if output.Status != sfntypes.ExecutionStatusRunning {
				log.Printf("[warn] already stopped execution: %s", executionArn)
				return result, ctx.Err()
			}
			_, err := svc.client.StopExecution(stopCtx, &sfn.StopExecutionInput{
				ExecutionArn: aws.String(executionArn),
				Error:        aws.String("stefunny.ContextCanceled"),
				Cause:        aws.String(ctx.Err().Error()),
			})
			if err != nil {
				log.Printf("[error] stop execution failed: %s", err.Error())
				return result, ctx.Err()
			}
			return result, ctx.Err()
		case <-ticker.C:
		}
		output, err = svc.client.DescribeExecution(ctx, input)
		if err != nil {
			return nil, err
		}
	}
	log.Printf("[info] execution status: %s", output.Status)
	result := &WaitExecutionOutput{
		Success:   output.Status == sfntypes.ExecutionStatusSucceeded,
		Failed:    output.Status == sfntypes.ExecutionStatusFailed,
		StartDate: *output.StartDate,
		StopDate:  *output.StopDate,
	}
	if output.Output != nil {
		result.Output = *output.Output
	}
	historyOutput, err := svc.client.GetExecutionHistory(ctx, &sfn.GetExecutionHistoryInput{
		ExecutionArn:         aws.String(executionArn),
		IncludeExecutionData: aws.Bool(true),
		MaxResults:           5,
		ReverseOrder:         true,
	})
	if err != nil {
		return nil, err
	}
	for _, event := range historyOutput.Events {
		if event.Type == sfntypes.HistoryEventTypeExecutionAborted {
			result.Datail = event.ExecutionAbortedEventDetails
			break
		}
		if event.Type == sfntypes.HistoryEventTypeExecutionFailed {
			result.Datail = event.ExecutionFailedEventDetails
			break
		}
		if event.Type == sfntypes.HistoryEventTypeExecutionTimedOut {
			result.Datail = event.ExecutionTimedOutEventDetails
			break
		}
	}
	return result, nil
}

type HistoryEvent struct {
	StartDate time.Time
	Step      string
	sfntypes.HistoryEvent
}

func (svc *SFnServiceImpl) GetExecutionHistory(ctx context.Context, executionArn string) ([]HistoryEvent, error) {
	describeOutput, err := svc.client.DescribeExecution(ctx, &sfn.DescribeExecutionInput{
		ExecutionArn: aws.String(executionArn),
	})
	if err != nil {
		return nil, err
	}
	p := sfn.NewGetExecutionHistoryPaginator(svc.client, &sfn.GetExecutionHistoryInput{
		ExecutionArn:         aws.String(executionArn),
		IncludeExecutionData: aws.Bool(true),
		MaxResults:           100,
	})
	events := make([]HistoryEvent, 0)
	var step string
	for p.HasMorePages() {
		output, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, event := range output.Events {
			if event.StateEnteredEventDetails != nil {
				step = *event.StateEnteredEventDetails.Name
			}
			events = append(events, HistoryEvent{
				StartDate:    *describeOutput.StartDate,
				Step:         step,
				HistoryEvent: event,
			})

		}
	}
	return events, nil
}

func (event HistoryEvent) Elapsed() time.Duration {
	return event.HistoryEvent.Timestamp.Sub(event.StartDate)
}

func (svc *SFnServiceImpl) StartSyncExecution(ctx context.Context, stateMachine *StateMachine, executionName, input string) (*sfn.StartSyncExecutionOutput, error) {

	if executionName == "" {
		uuidObj, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		executionName = uuidObj.String()
	}
	output, err := svc.client.StartSyncExecution(ctx, &sfn.StartSyncExecutionInput{
		StateMachineArn: stateMachine.StateMachineArn,
		Input:           aws.String(input),
		Name:            aws.String(executionName),
		TraceHeader:     aws.String(*stateMachine.Name + "_" + executionName),
	})
	if err != nil {
		return nil, err
	}
	return output, nil
}
