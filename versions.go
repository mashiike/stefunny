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

// JSON renders f.Data.Versions as an indented JSON string.
// Returns an error if marshaling or indenting fails.
func (f OutputFormatter) JSON() (string, error) {
	if f.Data == nil {
		return "[]", nil
	}
	b, err := json.Marshal(f.Data.Versions)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	var out bytes.Buffer
	if err := json.Indent(&out, b, "", "  "); err != nil {
		return "", fmt.Errorf("failed to indent JSON: %w", err)
	}
	return out.String(), nil
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

// Table renders f.Data.Versions as a table string.
// Returns an error if the underlying tablewriter fails to append rows or render.
func (f OutputFormatter) Table() (string, error) {
	buf := new(strings.Builder)
	w := tablewriter.NewTable(buf)
	w.Header("Version", "Aliases", "Creation Date", "Revision ID", "Description")
	for _, v := range f.Data.Versions {
		if err := w.Append(
			fmt.Sprintf("%d", v.Version),
			strings.Join(v.Aliases, ","),
			v.CreationDate.Local().Format(time.RFC3339),
			v.RevisionID,
			v.Description,
		); err != nil {
			return "", fmt.Errorf("failed to append version row: %w", err)
		}
	}
	if err := w.Render(); err != nil {
		return "", fmt.Errorf("failed to render versions table: %w", err)
	}
	return buf.String(), nil
}

// Render returns f's string representation according to f.Format ("json", "tsv", or table by default).
// Returns an error if the table format fails to render (see Table).
//
// Named Render rather than String because f can fail to format and therefore
// does not satisfy fmt.Stringer.
func (f OutputFormatter) Render() (string, error) {
	switch f.Format {
	case "json":
		return f.JSON()
	case "tsv":
		return f.TSV(), nil
	default:
		return f.Table()
	}
}

// Versions lists the deployed versions of the state machine, optionally purging older ones.
// Returns an error if describing, listing, or (when opt.Delete is set) purging the versions fails.
func (app *App) Versions(ctx context.Context, opt VersionsOption) error {
	sfnSvc, err := app.sfnService(ctx)
	if err != nil {
		return err
	}
	stateMachine, err := sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name: app.cfg.StateMachineName(),
	})
	if err != nil {
		if !errors.Is(err, ErrStateMachineDoesNotExist) {
			return fmt.Errorf("failed to describe current state machine status: %w", err)
		}
		log.Println("[info] State machine does not exist")
		return nil
	}
	if opt.Delete {
		if err := sfnSvc.PurgeStateMachineVersions(ctx, stateMachine, opt.KeepVersions); err != nil {
			return fmt.Errorf("failed to delete older versions: %w", err)
		}
	}
	versions, err := sfnSvc.ListStateMachineVersions(ctx, stateMachine)
	if err != nil {
		return fmt.Errorf("failed to list state machine versions: %w", err)
	}
	formatter := &OutputFormatter{
		Data:   versions,
		Format: opt.Format,
	}
	s, err := formatter.Render()
	if err != nil {
		return fmt.Errorf("failed to format versions: %w", err)
	}
	fmt.Println(s)
	return nil
}
