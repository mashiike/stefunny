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

// DiffString renders the diff between s (the current state, possibly nil)
// and newSchedule (the desired state). opt.Ignore excludes matching paths
// from the comparison. Returns an error if opt.Ignore is an invalid jq
// query.
func (s *Schedule) DiffString(newSchedule *Schedule, opt DiffStringOption) (string, error) {
	from := s.Source()
	to := newSchedule.Source()

	ds, err := JSONDiffString(
		s.configureJSON(), newSchedule.configureJSON(),
		JSONDiffFromURI(from),
		JSONDiffToURI(to),
		JSONDiffUnified(opt.Unified),
		JSONDiffIgnore(opt.Ignore),
	)
	if err != nil {
		return "", fmt.Errorf("diff schedule: %w", err)
	}
	return ds, nil
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

// DiffString renders the diff between s (the current state) and
// newSchedules (the desired state), matched by name. opt.Ignore excludes
// matching paths from each schedule's comparison. Returns an error if
// opt.Ignore is an invalid jq query.
func (s Schedules) DiffString(newSchedules Schedules, opt DiffStringOption) (string, error) {
	result := sliceDiff(s, newSchedules, func(schedule *Schedule) string {
		return coalesce(schedule.Name)
	})
	var builder strings.Builder
	var zero *Schedule
	for _, schedule := range result.Delete {
		ds, err := schedule.DiffString(zero, opt)
		if err != nil {
			return "", err
		}
		builder.WriteString(ds)
		builder.WriteRune('\n')
	}
	for _, change := range result.Change {
		ds, err := change.Before.DiffString(change.After, opt)
		if err != nil {
			return "", err
		}
		builder.WriteString(ds)
		builder.WriteRune('\n')
	}
	for _, schedule := range result.Add {
		ds, err := zero.DiffString(schedule, opt)
		if err != nil {
			return "", err
		}
		builder.WriteString(ds)
		builder.WriteRune('\n')
	}
	return builder.String(), nil
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
