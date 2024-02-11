package stefunny

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

type ExecuteOption struct {
	Stdout io.Writer `kong:"-" json:"-"`
	Stderr io.Writer `kong:"-" json:"-"`

	Input         string  `name:"input" help:"input JSON string" default:"-" type:"existingfile" json:"input,omitempty"`
	ExecutionName string  `name:"name" help:"execution name" default:"" json:"name,omitempty"`
	Async         bool    `name:"async" help:"start execution and return immediately" json:"async,omitempty"`
	DumpHistory   bool    `name:"dump-history" help:"dump execution history" json:"dump_history,omitempty"`
	Qualifier     *string `name:"qualifier" help:"state machine version qualifier" json:"qualifier,omitempty"`
}

func (app *App) Execute(ctx context.Context, opt ExecuteOption) error {
	var inputReader io.Reader
	if opt.Input == "-" {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			defaultInput := `{"Comment": "Insert your JSON here"}`
			log.Println("[warn] no input is specified, so we'll use the default input in .")
			inputReader = strings.NewReader(defaultInput)
		} else {
			inputReader = os.Stdin
		}
	} else {
		fp, err := os.Open(opt.Input)
		if err != nil {
			return err
		}
		defer fp.Close()
		inputReader = fp
	}
	dec := json.NewDecoder(inputReader)
	var inputJSON interface{}
	if err := dec.Decode(&inputJSON); err != nil {
		return err
	}
	bs, err := json.MarshalIndent(inputJSON, "", "  ")
	if err != nil {
		return err
	}
	input := string(bs)
	log.Printf("[info] input:\n%s\n", input)
	stateMachine, err := app.sfnSvc.DescribeStateMachine(ctx, &DescribeStateMachineInput{
		Name: app.cfg.StateMachineName(),
	})
	if err != nil {
		return err
	}
	output, err := app.sfnSvc.StartExecution(ctx, stateMachine, &StartExecutionInput{
		Input:         input,
		ExecutionName: opt.ExecutionName,
		Qualifier:     opt.Qualifier,
		Async:         opt.Async,
	})
	if err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}
	if opt.Async {
		return nil
	}
	log.Printf("[info] execution time: %s", output.Elapsed())
	if !opt.DumpHistory {
		return nil
	}
	if output.CanNotDumpHistory {
		log.Println("[warn] this state machine can not dump history.")
		return nil
	}
	events, err := app.sfnSvc.GetExecutionHistory(ctx, output.ExecutionArn)
	if err != nil {
		return err
	}
	table := tablewriter.NewWriter(opt.Stderr)
	table.SetHeader([]string{"ID", "Type", "Step", "Elapsed(ms)", "Timestamp"})
	for _, event := range events {
		table.Append([]string{
			fmt.Sprintf("%3d", event.Id),
			fmt.Sprintf("%v", event.HistoryEvent.Type),
			event.Step,
			fmt.Sprintf("%d", event.Elapsed().Milliseconds()),
			event.Timestamp.Format(time.RFC3339),
		})
	}
	table.Render()

	if output.Datail != nil {
		log.Printf("[info] execution detail:\n%s", MarshalJSONString(output.Datail))
	}
	if output.Failed != nil && *output.Failed {
		return errors.New("state machine execution failed")
	}
	log.Printf("[info] execution success")
	if opt.Stdout != nil && output.Output != nil && len(*output.Output) > 0 {
		io.WriteString(opt.Stdout, *output.Output)
		io.WriteString(opt.Stdout, "\n")
	}
	return nil
}
