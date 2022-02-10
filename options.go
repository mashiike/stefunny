package stefunny

import (
	"io"
)

const dryRunStr = "DRY RUN"

type DeleteOption struct {
	DryRun bool
	Force  bool
}

func (opt DeleteOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

type DeployOption struct {
	DryRun                 bool
	ScheduleEnabled        *bool
	SkipDeployStateMachine bool
}

func (opt DeployOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

type RenderOption struct {
	Writer io.Writer
	Format string
}

type LoadConfigOption struct {
	TFState string
	ExtStr  map[string]string
	ExtCode map[string]string
}

type ExecuteOption struct {
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	ExecutionName string
	Async         bool
	DumpHistory   bool
}
