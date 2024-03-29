package stefunny

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/scheduler"
	schedulertypes "github.com/aws/aws-sdk-go-v2/service/scheduler/types"
)

type Schedule struct {
	scheduler.CreateScheduleInput
	ScheduleArn     *string    `min:"1" type:"string"`
	CreationDate    *time.Time `type:"timestamp"`
	ConfigFilePath  *string
	ConfigFileIndex int
}

func (s *Schedule) Source() string {
	if s == nil {
		return knownAfterDeployArn
	}
	if s.ScheduleArn != nil {
		return *s.ScheduleArn
	}
	if s.ConfigFilePath != nil {
		return fmt.Sprintf("trigger.schedule[%d] in %s", s.ConfigFileIndex, *s.ConfigFilePath)
	}
	if s.Name != nil {
		return *s.Name
	}
	return knownAfterDeployArn
}

func (s *Schedule) SetStateMachineQualifiedArn(stateMachineArn string) {
	if s.Target == nil {
		s.Target = &schedulertypes.Target{}
	}
	s.Target.Arn = &stateMachineArn
}

func (s *Schedule) configureJSON() string {
	if s == nil {
		return "null"
	}
	return MarshalJSONString(s.CreateScheduleInput, map[string]interface{}{
		"Target": s.Target,
	})
}

func (s *Schedule) HasItPassed() bool {
	if s.EndDate != nil {
		log.Printf("[debug] check if schedule `%s` has passed, end_date=%s", coalesce(s.Name), s.EndDate.String())
		if time.Now().After(*s.EndDate) {
			return true
		}
	}
	// ScheduleExpressionが at(yyyy-mm-ddThh:mm:ss) の場合は、時刻をパースして現在時刻と比較する
	expression := coalesce(s.ScheduleExpression)
	if strings.HasPrefix(expression, "at(") {
		at := expression[3 : len(expression)-1]
		tz := coalesce(s.ScheduleExpressionTimezone)
		var loc *time.Location
		if tz == "" {
			loc = time.UTC
		} else {
			var err error
			loc, err = time.LoadLocation(tz)
			if err != nil {
				log.Printf("[warn] failed to load location `%s` as : %s", tz, err)
				return false
			}
		}
		t, err := time.Parse("2006-01-02T15:04:05", at)
		if err != nil {
			log.Printf("[warn] failed to parse schedule expression `%s` as : %s", expression, err)
			return false
		}
		log.Printf("[debug] check if schedule `%s` has passed, at=%s tz=%s", coalesce(s.Name), t.String(), loc.String())
		t = t.In(loc)
		now := time.Now().In(loc)
		return now.After(t)
	}
	return false
}

func (s *Schedule) String() string {
	var builder strings.Builder
	builder.WriteString(colorRestString(s.configureJSON()))
	return builder.String()
}

func (s *Schedule) DiffString(newSchedule *Schedule, unified bool) string {
	var builder strings.Builder
	from := s.Source()
	to := newSchedule.Source()

	builder.WriteString(JSONDiffString(
		s.configureJSON(), newSchedule.configureJSON(),
		JSONDiffFromURI(from),
		JSONDiffToURI(to),
		JSONDiffUnified(unified),
	))
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

func (s Schedules) SetStateMachineQualifiedArn(stateMachineArn string) {
	for _, schedule := range s {
		schedule.SetStateMachineQualifiedArn(stateMachineArn)
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

func (s Schedules) DiffString(newSchedules Schedules, unified bool) string {
	result := sliceDiff(s, newSchedules, func(schedule *Schedule) string {
		return coalesce(schedule.Name)
	})
	var builder strings.Builder
	var zero *Schedule
	for _, schedule := range result.Delete {
		builder.WriteString(schedule.DiffString(zero, unified))
		builder.WriteRune('\n')
	}
	for _, change := range result.Change {
		builder.WriteString(change.Before.DiffString(change.After, unified))
		builder.WriteRune('\n')
	}
	for _, schedule := range result.Add {
		builder.WriteString(zero.DiffString(schedule, unified))
		builder.WriteRune('\n')
	}
	return builder.String()
}

func (s Schedules) FilterPassed() (result, passed Schedules) {
	for _, schedule := range s {
		if !schedule.HasItPassed() {
			result = append(result, schedule)
		} else {
			passed = append(passed, schedule)
		}
	}
	return result, passed
}

func (s Schedules) Names() []string {
	names := make([]string, 0, len(s))
	for _, schedule := range s {
		if name := coalesce(schedule.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func (s Schedules) FindByName(name string) (*Schedule, bool) {
	for _, schedule := range s {
		if coalesce(schedule.Name) == name {
			return schedule, true
		}
	}
	return nil, false
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
