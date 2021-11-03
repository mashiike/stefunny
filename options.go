package sffle

import "io"

const dryRunStr = "DRY RUN"

type DeployOption struct {
	DryRun bool
}

func (opt DeployOption) DryRunString() string {
	if opt.DryRun {
		return dryRunStr
	}
	return ""
}

type RenderOption struct {
	Writer io.Writer
}
