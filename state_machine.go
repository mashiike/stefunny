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
	StateMachineArn *string
	Status          sfntypes.StateMachineStatus
}

func (s *StateMachine) QualifiedARN(name string) string {
	if name == "" {
		return coalesce(s.StateMachineArn)
	}
	return fmt.Sprintf("%s:%s", coalesce(s.StateMachineArn), name)
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
