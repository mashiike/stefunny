package stefunny

import "io"

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
}
