package stefunny

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/mashiike/stefunny/internal/sfnx"
	"github.com/shogo82148/go-retry"
)

const (
	defaultAliasName = "current"
)

var (
	ErrStateMachineDoesNotExist = errors.New("state machine does not exist")
	ErrRollbackTargetNotFound   = errors.New("rollback target not found")
)

type SFnClient interface {
	sfn.ListStateMachinesAPIClient
	sfnx.ListStateMachineAliasesAPIClient
	sfnx.ListStateMachineVersionsAPIClient
	CreateStateMachine(ctx context.Context, params *sfn.CreateStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineOutput, error)
	CreateStateMachineAlias(ctx context.Context, params *sfn.CreateStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.CreateStateMachineAliasOutput, error)
	DescribeStateMachine(ctx context.Context, params *sfn.DescribeStateMachineInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error)
	DescribeStateMachineAlias(ctx context.Context, params *sfn.DescribeStateMachineAliasInput, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineAliasOutput, error)
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

type SFnService interface {
	DescribeStateMachine(ctx context.Context, name string, optFns ...func(*sfn.Options)) (*StateMachine, error)
	GetStateMachineArn(ctx context.Context, name string, optFns ...func(*sfn.Options)) (string, error)
	DeployStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*DeployStateMachineOutput, error)
	DeleteStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error
	RollbackStateMachine(ctx context.Context, stateMachine *StateMachine, keepVersion bool, dryRun bool, optFns ...func(*sfn.Options)) error
	ListStateMachineVersions(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*ListStateMachineVersionsOutput, error)
	PurgeStateMachineVersions(ctx context.Context, stateMachine *StateMachine, keepVersions int, optFns ...func(*sfn.Options)) error
	StartExecution(ctx context.Context, stateMachine *StateMachine, params *StartExecutionInput, optFns ...func(*sfn.Options)) (*StartExecutionOutput, error)
	GetExecutionHistory(ctx context.Context, executionArn string) ([]HistoryEvent, error)
	SetAliasName(aliasName string)
}

type SFnServiceImpl struct {
	client                               SFnClient
	aliasName                            string
	cacheStateMachineArnByName           map[string]string
	cacheStateMachineAliasByAliasARN     map[string]*sfn.DescribeStateMachineAliasOutput
	cacheStateMachineVersionByVersionARN map[string]*sfn.DescribeStateMachineOutput
	cacheStateMachineVersionsByARN       map[string][]sfntypes.StateMachineVersionListItem
	cacheStateMachineAliasesByARN        map[string][]sfntypes.StateMachineAliasListItem
	retryPolicy                          retry.Policy
}

var _ SFnService = (*SFnServiceImpl)(nil)

func NewSFnService(client SFnClient) *SFnServiceImpl {
	return &SFnServiceImpl{
		client:                               client,
		aliasName:                            defaultAliasName,
		cacheStateMachineArnByName:           make(map[string]string),
		cacheStateMachineAliasByAliasARN:     make(map[string]*sfn.DescribeStateMachineAliasOutput),
		cacheStateMachineVersionByVersionARN: make(map[string]*sfn.DescribeStateMachineOutput),
		cacheStateMachineVersionsByARN:       make(map[string][]sfntypes.StateMachineVersionListItem),
		cacheStateMachineAliasesByARN:        make(map[string][]sfntypes.StateMachineAliasListItem),
		retryPolicy: retry.Policy{
			MinDelay: time.Second,
			MaxDelay: 10 * time.Second,
			MaxCount: 30,
		},
	}
}

func (svc *SFnServiceImpl) SetAliasName(aliasName string) {
	svc.aliasName = aliasName
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
	svc.cacheStateMachineArnByName[coalesce(stateMachine.Name)] = *output.StateMachineArn
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

func (svc *SFnServiceImpl) describeStateMachineAlias(ctx context.Context, aliasARN string, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineAliasOutput, error) {
	if alias, ok := svc.cacheStateMachineAliasByAliasARN[aliasARN]; ok {
		return alias, nil
	}
	alias, err := svc.client.DescribeStateMachineAlias(ctx, &sfn.DescribeStateMachineAliasInput{
		StateMachineAliasArn: aws.String(aliasARN),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	svc.cacheStateMachineAliasByAliasARN[aliasARN] = alias
	return alias, nil
}

func (svc *SFnServiceImpl) describeStateMachineVersion(ctx context.Context, versionARN string, optFns ...func(*sfn.Options)) (*sfn.DescribeStateMachineOutput, error) {
	if version, ok := svc.cacheStateMachineVersionByVersionARN[versionARN]; ok {
		return version, nil
	}
	version, err := svc.client.DescribeStateMachine(ctx, &sfn.DescribeStateMachineInput{
		StateMachineArn: aws.String(versionARN),
	}, optFns...)
	if err != nil {
		return nil, err
	}
	svc.cacheStateMachineVersionByVersionARN[versionARN] = version
	return version, nil
}

func (svc *SFnServiceImpl) updateCurrentArias(ctx context.Context, stateMachine *StateMachine, versionARN string, optFns ...func(*sfn.Options)) error {
	aliasARN := stateMachine.QualifiedARN(svc.aliasName)
	alias, err := svc.describeStateMachineAlias(ctx, aliasARN, optFns...)
	if err != nil {
		var notExists *sfntypes.ResourceNotFound
		if errors.As(err, &notExists) {
			log.Println("[info] current alias does not exist, create it...")
			output, err := svc.client.CreateStateMachineAlias(ctx, &sfn.CreateStateMachineAliasInput{
				Name: aws.String(svc.aliasName),
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

type ListStateMachineVersionsOutput struct {
	StateMachineArn string
	Versions        []StateMachineVersionListItem
}

type StateMachineVersionListItem struct {
	StateMachineVersionARN string
	Version                int       `json:"version"`
	Aliases                []string  `json:"aliases,omitempty"`
	Description            string    `json:"description,omitempty"`
	CreationDate           time.Time `json:"creation_date"`
	RevisionID             string    `json:"revision_id,omitempty"`
}

func (svc *SFnServiceImpl) ListStateMachineVersions(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*ListStateMachineVersionsOutput, error) {
	return svc.listStateMachineVersions(ctx, stateMachine, optFns...)
}

func (svc *SFnServiceImpl) listStateMachineVersions(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) (*ListStateMachineVersionsOutput, error) {
	var ok bool
	var aliasListItemes []sfntypes.StateMachineAliasListItem
	if aliasListItemes, ok = svc.cacheStateMachineAliasesByARN[coalesce(stateMachine.StateMachineArn)]; !ok {
		p := sfnx.NewListStateMachineAliasesPaginator(svc.client, &sfn.ListStateMachineAliasesInput{
			StateMachineArn: stateMachine.StateMachineArn,
			MaxResults:      32,
		})
		aliasListItemes = make([]sfntypes.StateMachineAliasListItem, 0)
		for p.HasMorePages() {
			output, err := p.NextPage(ctx, optFns...)
			if err != nil {
				return nil, fmt.Errorf("list state machine aliases failed: %w", err)
			}
			aliasListItemes = append(aliasListItemes, output.StateMachineAliases...)
		}
		svc.cacheStateMachineAliasesByARN[coalesce(stateMachine.StateMachineArn)] = aliasListItemes
	}
	aliasesByVersionARN := make(map[string][]string, len(aliasListItemes))
	for _, item := range aliasListItemes {
		alias, err := svc.describeStateMachineAlias(ctx, *item.StateMachineAliasArn, optFns...)
		if err != nil {
			return nil, fmt.Errorf("describe state machine alias failed: %w", err)
		}
		for _, routing := range alias.RoutingConfiguration {
			aliasesByVersionARN[*routing.StateMachineVersionArn] = append(aliasesByVersionARN[*routing.StateMachineVersionArn], *alias.Name)
		}
	}

	var versionListItems []sfntypes.StateMachineVersionListItem
	if versionListItems, ok = svc.cacheStateMachineVersionsByARN[coalesce(stateMachine.StateMachineArn)]; !ok {
		p := sfnx.NewListStateMachineVersionsPaginator(svc.client, &sfn.ListStateMachineVersionsInput{
			StateMachineArn: stateMachine.StateMachineArn,
			MaxResults:      32,
		})
		versionListItems = make([]sfntypes.StateMachineVersionListItem, 0)
		for p.HasMorePages() {
			output, err := p.NextPage(ctx, optFns...)
			if err != nil {
				return nil, err
			}
			versionListItems = append(versionListItems, output.StateMachineVersions...)
		}
		svc.cacheStateMachineVersionsByARN[coalesce(stateMachine.StateMachineArn)] = versionListItems
	}
	output := &ListStateMachineVersionsOutput{
		StateMachineArn: coalesce(stateMachine.StateMachineArn),
		Versions:        make([]StateMachineVersionListItem, 0, len(versionListItems)),
	}
	for _, item := range versionListItems {
		versionNumber, err := extructVersion(*item.StateMachineVersionArn)
		if err != nil {
			log.Printf("[warn] extruct version `%s` failed: %s", *item.StateMachineVersionArn, err)
			continue
		}
		versionDetail, err := svc.describeStateMachineVersion(ctx, *item.StateMachineVersionArn, optFns...)
		if err != nil {
			log.Printf("[warn] describe version `%s` failed: %s", *item.StateMachineVersionArn, err)
			continue
		}
		version := &StateMachineVersionListItem{
			StateMachineVersionARN: *item.StateMachineVersionArn,
			Version:                versionNumber,
			CreationDate:           *item.CreationDate,
			Aliases:                aliasesByVersionARN[*item.StateMachineVersionArn],
			RevisionID:             coalesce(versionDetail.RevisionId),
			Description:            coalesce(versionDetail.Description),
		}
		output.Versions = append(output.Versions, *version)
	}
	// sort by version number desc
	sort.Slice(output.Versions, func(i, j int) bool {
		return output.Versions[i].Version > output.Versions[j].Version
	})
	return output, nil
}

func (svc *SFnServiceImpl) PurgeStateMachineVersions(ctx context.Context, stateMachine *StateMachine, keepVerions int, optFns ...func(*sfn.Options)) error {
	if stateMachine.StateMachineArn == nil {
		return ErrStateMachineDoesNotExist
	}
	if keepVerions < 1 {
		log.Println("[info] keep version is less than 1, skip purge")
		return nil
	}
	output, err := svc.listStateMachineVersions(ctx, stateMachine, optFns...)
	if err != nil {
		return fmt.Errorf("list state machine versions failed: %w", err)
	}
	errs := make([]error, 0, len(output.Versions))
	for i, v := range output.Versions {
		if i == 0 {
			log.Printf("[info] keep latest version `%d`", v.Version)
			continue
		}
		if i < keepVerions {
			log.Printf("[debug] keep version `%d`", v.Version)
			continue
		}
		if len(v.Aliases) > 0 {
			log.Printf("[warn] version `%d` has aliases [%s], skip delete", v.Version, strings.Join(v.Aliases, ","))
			continue
		}
		log.Printf("[info] deleting state machine version %d (`%s`)", v.Version, v.StateMachineVersionARN)
		err := svc.deleteStateMachineVersion(ctx, v.StateMachineVersionARN, optFns...)
		if err != nil {
			errs = append(errs, fmt.Errorf("deleting version `%d` failed: %w", v.Version, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("delete versions failed: %w", errors.Join(errs...))
	}
	return nil
}

func (svc *SFnServiceImpl) RollbackStateMachine(ctx context.Context, stateMachine *StateMachine, keepVersion bool, dryRun bool, optFns ...func(*sfn.Options)) error {
	if stateMachine.StateMachineArn == nil {
		return ErrStateMachineDoesNotExist
	}
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s already deleting...\n", coalesce(stateMachine.StateMachineArn))
		return nil
	}
	aliasARN := stateMachine.QualifiedARN(svc.aliasName)
	alias, err := svc.describeStateMachineAlias(ctx, aliasARN, optFns...)
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
	output, err := svc.listStateMachineVersions(ctx, stateMachine, optFns...)
	if err != nil {
		return fmt.Errorf("list state machine versions failed: %w", err)
	}
	targetVersion := 0
	targetVersionItem := StateMachineVersionListItem{
		StateMachineVersionARN: currentVersionArn,
	}

	for _, v := range output.Versions {
		log.Println("[debug] found version: ", v.StateMachineVersionARN)
		if v.Version >= currentVersion {
			log.Println("[debug] skip version: ", v.Version)
			continue
		}
		if targetVersion < v.Version {
			targetVersion = v.Version
			targetVersionItem = v
		}
	}
	log.Println("[debug] target version: ", targetVersion)
	if targetVersionItem.StateMachineVersionARN == currentVersionArn {
		log.Println("[notice] no previous version found, can not rollback")
		return ErrRollbackTargetNotFound
	}
	log.Printf("[info] rollback to version `%d`", targetVersion)
	if !dryRun {
		if err := svc.updateCurrentArias(ctx, stateMachine, targetVersionItem.StateMachineVersionARN, optFns...); err != nil {
			return fmt.Errorf("update current alias failed: %w", err)
		}
		log.Println("[info] rollback success")
	}
	if keepVersion {
		return nil
	}
	if len(targetVersionItem.Aliases) > 0 {
		log.Printf("[warn] version `%d` has aliases [%s], skip delete", targetVersion, strings.Join(targetVersionItem.Aliases, ","))
		return nil
	}
	log.Printf("[info] deleting version `%d`", currentVersion)
	if !dryRun {
		err = svc.deleteStateMachineVersion(ctx, currentVersionArn, optFns...)
		if err != nil {
			return fmt.Errorf("delete version failed: %w", err)
		}
		log.Printf("[info] `%s` deleted", currentVersionArn)
	}
	return nil
}

func (svc *SFnServiceImpl) deleteStateMachineVersion(ctx context.Context, versionARN string, optFns ...func(*sfn.Options)) error {
	retrier := svc.retryPolicy.Start(ctx)
	for retrier.Continue() {
		_, err := svc.client.DeleteStateMachineVersion(ctx, &sfn.DeleteStateMachineVersionInput{
			StateMachineVersionArn: &versionARN,
		}, optFns...)
		if err == nil {
			return nil
		}
		var apiErr smithy.APIError
		if !errors.As(err, &apiErr) {
			log.Printf("[debug] unexpected error: %s", err)
			return err
		}
		if apiErr.ErrorCode() == "ConflictException" {
			log.Printf("[debug] conflict error: %s", err)
			errStr := err.Error()
			if !strings.Contains(errStr, "Current list of aliases referencing this version: [") {
				return err
			}
			i := strings.Index(errStr, "[")
			j := strings.Index(errStr, "]")
			if i == -1 || j == -1 {
				return err
			}
			aliases := strings.Split(errStr[i+1:j], ",")
			found := false
			for _, alias := range aliases {
				if strings.Contains(alias, svc.aliasName) {
					found = true
					break
				}
			}
			if !found {
				log.Printf("[warn] `%s` is referenced by other alias [%s], skip delete", versionARN, strings.Join(aliases, ","))
				return nil
			}
			continue
		}
		return err
	}
	return errors.New("max retry count exceeded")
}

func (svc *SFnServiceImpl) DeleteStateMachine(ctx context.Context, stateMachine *StateMachine, optFns ...func(*sfn.Options)) error {
	if stateMachine.Status == sfntypes.StateMachineStatusDeleting {
		log.Printf("[info] %s already deleting...\n", coalesce(stateMachine.StateMachineArn))
		return nil
	}
	retirer := svc.retryPolicy.Start(ctx)
	for retirer.Continue() {
		_, err := svc.client.DeleteStateMachine(ctx, &sfn.DeleteStateMachineInput{
			StateMachineArn: stateMachine.StateMachineArn,
		}, optFns...)
		if err == nil {
			return nil
		}
		var apiErr smithy.APIError
		if !errors.As(err, &apiErr) {
			log.Printf("[debug] unexpected error: %s", err.Error())
			return err
		}
		if apiErr.ErrorCode() != "ConflictException" {
			log.Printf("[debug] conflict error: %s", err.Error())
			continue
		}
		return err
	}
	return errors.New("max retry count exceeded")
}

type StartExecutionInput struct {
	ExecutionName string
	Input         string
	Qualifier     *string
	Target        string
	Async         bool
}

type StartExecutionOutput struct {
	CanNotDumpHistory bool
	ExecutionArn      string
	StartDate         time.Time
	Success           *bool
	Failed            *bool
	StopDate          *time.Time
	Output            *string
	Datail            interface{}
}

func (o *StartExecutionOutput) Elapsed() time.Duration {
	if o.StopDate == nil {
		return -1
	}
	return o.StopDate.Sub(o.StartDate)
}

func (svc *SFnServiceImpl) StartExecution(ctx context.Context, stateMachine *StateMachine, params *StartExecutionInput, optFns ...func(*sfn.Options)) (*StartExecutionOutput, error) {
	if params.ExecutionName == "" {
		uuidObj, err := uuid.NewRandom()
		if err != nil {
			return nil, err
		}
		params.ExecutionName = uuidObj.String()
	}
	params.Target = stateMachine.QualifiedARN(coalesce(params.Qualifier))
	switch stateMachine.Type {
	case sfntypes.StateMachineTypeStandard:
		return svc.startExecutionForStandard(ctx, stateMachine, params, optFns...)
	case sfntypes.StateMachineTypeExpress:
		output, err := svc.startExecutionForExpress(ctx, stateMachine, params, optFns...)
		if err != nil {
			return nil, err
		}
		output.CanNotDumpHistory = true
		return output, nil
	default:
		return nil, fmt.Errorf("unknown state machine type: %s", stateMachine.Type)
	}
}

func (svc *SFnServiceImpl) startExecutionForStandard(ctx context.Context, stateMachine *StateMachine, params *StartExecutionInput, _ ...func(*sfn.Options)) (*StartExecutionOutput, error) {
	output, err := svc.startExecution(ctx, stateMachine, params)
	if err != nil {
		return nil, err
	}
	log.Printf("[notice] execution arn=%s", output.ExecutionArn)
	log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
	if params.Async {
		return output, nil
	}
	waitOutput, err := svc.waitExecution(ctx, output.ExecutionArn)
	if err != nil {
		return output, err
	}
	output.Success = &waitOutput.Success
	output.Failed = &waitOutput.Failed
	output.StopDate = &waitOutput.StopDate
	output.Output = &waitOutput.Output
	output.Datail = waitOutput.Datail
	return output, nil
}

func (svc *SFnServiceImpl) startExecutionForExpress(ctx context.Context, stateMachine *StateMachine, params *StartExecutionInput, _ ...func(*sfn.Options)) (*StartExecutionOutput, error) {
	if params.Async {
		output, err := svc.startExecution(ctx, stateMachine, params)
		if err != nil {
			return nil, err
		}
		log.Printf("[notice] execution arn=%s", output.ExecutionArn)
		log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
		return output, nil
	}
	syncOutput, err := svc.client.StartSyncExecution(ctx, &sfn.StartSyncExecutionInput{
		StateMachineArn: &params.Target,
		Input:           aws.String(params.Input),
		Name:            aws.String(params.ExecutionName),
		TraceHeader:     aws.String(coalesce(stateMachine.Name) + "_" + params.ExecutionName),
	})
	if err != nil {
		return nil, err
	}
	succeeded := syncOutput.Status == sfntypes.SyncExecutionStatusSucceeded
	failed := syncOutput.Status == sfntypes.SyncExecutionStatusFailed
	output := &StartExecutionOutput{
		ExecutionArn: *syncOutput.ExecutionArn,
		StartDate:    *syncOutput.StartDate,
		Success:      &succeeded,
		Failed:       &failed,
		StopDate:     syncOutput.StopDate,
	}
	if syncOutput.Output != nil {
		output.Output = syncOutput.Output
	}
	if syncOutput.Status == sfntypes.SyncExecutionStatusFailed {
		output.Datail = sfntypes.ExecutionFailedEventDetails{
			Cause: syncOutput.Cause,
			Error: syncOutput.Error,
		}
	}
	return output, nil
}

func (svc *SFnServiceImpl) startExecution(ctx context.Context, stateMachine *StateMachine, params *StartExecutionInput) (*StartExecutionOutput, error) {
	output, err := svc.client.StartExecution(ctx, &sfn.StartExecutionInput{
		StateMachineArn: &params.Target,
		Input:           aws.String(params.Input),
		Name:            aws.String(params.ExecutionName),
		TraceHeader:     aws.String(coalesce(stateMachine.Name) + "_" + params.ExecutionName),
	})
	if err != nil {
		return nil, err
	}
	return &StartExecutionOutput{
		ExecutionArn: *output.ExecutionArn,
		StartDate:    *output.StartDate,
	}, nil
}

type waitExecutionOutput struct {
	Success   bool
	Failed    bool
	StartDate time.Time
	StopDate  time.Time
	Output    string
	Datail    interface{}
}

func (svc *SFnServiceImpl) waitExecution(ctx context.Context, executionArn string) (*waitExecutionOutput, error) {
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
			result := &waitExecutionOutput{
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
	result := &waitExecutionOutput{
		Success:   output.Status == sfntypes.ExecutionStatusSucceeded,
		Failed:    output.Status == sfntypes.ExecutionStatusFailed,
		StartDate: coalesce(output.StartDate),
		StopDate:  coalesce(output.StopDate),
		Output:    coalesce(output.Output),
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