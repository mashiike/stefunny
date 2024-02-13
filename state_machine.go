package stefunny

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
)

type StateMachine struct {
	sfn.CreateStateMachineInput
	CreationDate    *time.Time
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
	ConfigFilePath  *string
	DefinitionPath  *string
}

func (s *StateMachine) QualifiedArn(name string) string {
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

func (s *StateMachine) DiffString(newStateMachine *StateMachine, unified bool) string {
	var builder strings.Builder
	from := "[known after deploy]"
	if s != nil {
		from = coalesce(s.StateMachineArn, s.ConfigFilePath, s.Name)
	}
	to := "[known after deploy]"
	if newStateMachine != nil {
		to = coalesce(newStateMachine.StateMachineArn, newStateMachine.ConfigFilePath, newStateMachine.Name)
	}
	builder.WriteString(
		JSONDiffString(
			s.configureJSON(),
			newStateMachine.configureJSON(),
			JSONDiffUnified(unified),
			JSONDiffFromURI(from),
			JSONDiffToURI(to),
		),
	)
	def := "null"
	if s != nil {
		def = coalesce(s.Definition)
	}
	from = "[known after deploy]"
	if s != nil {
		from = coalesce(s.StateMachineArn, s.DefinitionPath, s.Name)
	}
	to = "[known after deploy]"
	if newStateMachine != nil {
		to = coalesce(newStateMachine.StateMachineArn, newStateMachine.DefinitionPath, newStateMachine.Name)
	}
	builder.WriteString(
		JSONDiffString(
			def,
			coalesce(newStateMachine.Definition),
			JSONDiffUnified(unified),
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
