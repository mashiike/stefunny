package stefunny

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
)

type VersionsOption struct {
	Format       string `help:"versions list format" default:"table" enum:"table,json,tsv" json:"format,omitempty"`
	Delete       bool   `help:"delete older versions" default:"false" json:"delete,omitempty"`
	KeepVersions int    `help:"Number of latest versions to keep. Older versions will be deleted with --delete." default:"5" json:"keep_versions,omitempty"`
}

type OutputFormatter struct {
	Data   *ListStateMachineVersionsOutput
	Format string
}

func (f OutputFormatter) JSON() string {
	if f.Data == nil {
		return "[]"
	}
	b, _ := json.Marshal(f.Data.Versions)
	var out bytes.Buffer
	json.Indent(&out, b, "", "  ")
	return out.String()
}

func (f OutputFormatter) TSV() string {
	buf := new(strings.Builder)
	for _, v := range f.Data.Versions {
		buf.WriteString(strings.Join([]string{
			fmt.Sprintf("%d", v.Version),
			strings.Join(v.Aliases, ","),
			v.CreationDate.Local().Format(time.RFC3339),
			v.RevisionID,
			v.Description,
		}, "\t") + "\n")
	}
	return buf.String()
}

func (f OutputFormatter) Table() string {
	buf := new(strings.Builder)
	w := tablewriter.NewWriter(buf)
	w.SetHeader([]string{"Version", "Aliases", "Creation Date", "Revision ID", "Description"})
	for _, v := range f.Data.Versions {
		w.Append([]string{
			fmt.Sprintf("%d", v.Version),
			strings.Join(v.Aliases, ","),
			v.CreationDate.Local().Format(time.RFC3339),
			v.RevisionID,
			v.Description,
		})
	}
	w.Render()
	return buf.String()
}

func (f OutputFormatter) String() string {
	switch f.Format {
	case "json":
		return f.JSON()
	case "tsv":
		return f.TSV()
	default:
		return f.Table()
	}
}

func (app *App) Versions(ctx context.Context, opt VersionsOption) error {
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, app.cfg.StateMachineName())
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
		log.Println("[info] State machine does not exist")
		return nil
	}
	if opt.Delete {
		if err := app.sfnSvc.PurgeStateMachineVersions(ctx, stateMachine, opt.KeepVersions); err != nil {
			return fmt.Errorf("failed to delete older versions: %w", err)
		}
	}
	versions, err := app.sfnSvc.ListStateMachineVersions(ctx, stateMachine)
	if err != nil {
		return fmt.Errorf("failed to list state machine versions: %w", err)
	}
	formatter := &OutputFormatter{
		Data:   versions,
		Format: opt.Format,
	}
	fmt.Println(formatter.String())
	return nil
}
