package stefunny

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

type Schedule struct {
	scheduler.CreateScheduleInput
	Arn          *string    `min:"1" type:"string"`
	CreationDate *time.Time `type:"timestamp"`
}

func (s *Schedule) SetStateMachineQualifiedARN(stateMachineArn string) {
	s.Target.Arn = &stateMachineArn
}

func (s *Schedule) configureJSON() string {
	return MarshalJSONString(s.CreateScheduleInput, map[string]interface{}{
		"Target": s.Target,
	})
}

func (s *Schedule) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString(s.configureJSON()))
	return builder.String()
}

func (s *Schedule) DiffString(newSchedule *Schedule) string {
	var builder strings.Builder
	builder.WriteString(colorRestString(JSONDiffString(s.configureJSON(), newSchedule.configureJSON())))
	return builder.String()
}

func (s *Schedule) SetEnabled(enabled bool) {
	if enabled {
		s.State = schedulertypes.ScheduleStateEnabled
	} else {
		s.State = schedulertypes.ScheduleStateDisabled
	}
}

type Schedules []*Schedule

func (s Schedules) SetStateMachineQualifiedARN(stateMachineArn string) {
	for _, schedule := range s {
		schedule.SetStateMachineQualifiedARN(stateMachineArn)
	}
}

func (s Schedules) String() string {
	var builder strings.Builder
	for _, schedule := range s {
		builder.WriteString(schedule.String())
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (s Schedules) SetEnabled(enabled bool) {
	for _, schedule := range s {
		schedule.SetEnabled(enabled)
	}
}

func (s Schedules) SyncState(other Schedules) {
	for _, schedule := range s {
		for _, otherSchedule := range other {
			if coalesce(schedule.Name) == coalesce(otherSchedule.Name) {
				schedule.State = otherSchedule.State
			}
		}
	}
}

func (s Schedules) DiffString(newSchedules Schedules) string {
	result := diff(s, newSchedules, func(schedule *Schedule) string {
		return coalesce(schedule.Name)
	})
	var builder strings.Builder
	for _, schedule := range result.Delete {
		builder.WriteString(colorRestString(JSONDiffString(schedule.configureJSON(), "null")))
		builder.WriteRune('\n')
	}
	for _, change := range result.Change {
		builder.WriteString(colorRestString(change.Before.DiffString(change.After)))
		builder.WriteRune('\n')
	}
	for _, schedule := range result.Add {
		builder.WriteString(colorRestString(JSONDiffString("null", schedule.configureJSON())))
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (s Schedules) Len() int {
	return len(s)
}

func (s Schedules) Less(i, j int) bool {
	return coalesce(s[i].Name) < coalesce(s[j].Name)
}

func (s Schedules) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
