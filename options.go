package sffle

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
