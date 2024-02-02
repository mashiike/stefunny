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

	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/mashiike/stefunny/internal/jsonutil"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

type ExecuteOption struct {
	Stdout io.Writer `kong:"-" json:"-"`
	Stderr io.Writer `kong:"-" json:"-"`

	Input         string `name:"input" help:"input JSON string" default:"-" type:"existingfile" json:"input,omitempty"`
	ExecutionName string `name:"name" help:"execution name" default:"" json:"name,omitempty"`
	Async         bool   `name:"async" help:"start execution and return immediately" json:"async,omitempty"`
	DumpHistory   bool   `name:"dump-history" help:"dump execution history" json:"dump_history,omitempty"`
}

func (app *App) Execute(ctx context.Context, opt *ExecuteOption) error {
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
	stateMachine, err := app.aws.DescribeStateMachine(ctx, app.cfg.StateMachine.Name)
	if err != nil {
		return err
	}
	if stateMachine.Type == sfntypes.StateMachineTypeExpress {
		return app.ExecuteForExpress(ctx, stateMachine, input, opt)
	}
	if stateMachine.Type == sfntypes.StateMachineTypeStandard {
		return app.ExecuteForStandard(ctx, stateMachine, input, opt)
	}
	return fmt.Errorf("unknown StateMachine Type:%s", stateMachine.Type)
}

func (app *App) ExecuteForExpress(ctx context.Context, stateMachine *StateMachine, input string, opt *ExecuteOption) error {
	if opt.DumpHistory {
		log.Println("[warn] this state machine is EXPRESS type, history is not supported.")
	}
	if opt.Async {
		output, err := app.aws.StartExecution(ctx, stateMachine, opt.ExecutionName, input)
		if err != nil {
			return err
		}
		log.Printf("[notice] execution arn=%s", output.ExecutionArn)
		log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
		return nil
	}
	output, err := app.aws.StartSyncExecution(ctx, stateMachine, opt.ExecutionName, input)
	if err != nil {
		return err
	}
	log.Printf("[notice] execution arn=%s", *output.ExecutionArn)
	log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))

	if output.Status != sfntypes.SyncExecutionStatusSucceeded {
		if output.Error != nil {
			log.Println("[info] error: ", *output.Error)
		}
		if output.Cause != nil {
			log.Println("[info] cause: ", *output.Cause)
		}
		return errors.New("state machine execution failed")
	}
	log.Printf("[info] execution success")
	if opt.Stdout != nil && output.Output != nil {
		io.WriteString(opt.Stdout, *output.Output)
		io.WriteString(opt.Stdout, "\n")
	}
	return nil
}

func (app *App) ExecuteForStandard(ctx context.Context, stateMachine *StateMachine, input string, opt *ExecuteOption) error {
	output, err := app.aws.StartExecution(ctx, stateMachine, opt.ExecutionName, input)
	if err != nil {
		return err
	}
	log.Printf("[notice] execution arn=%s", output.ExecutionArn)
	log.Printf("[notice] state at=%s", output.StartDate.In(time.Local))
	if opt.Async {
		return nil
	}
	waitOutput, err := app.aws.WaitExecution(ctx, output.ExecutionArn)
	if err != nil {
		return err
	}
	log.Printf("[info] execution time: %s", waitOutput.Elapsed())
	if opt.DumpHistory {
		events, err := app.aws.GetExecutionHistory(ctx, output.ExecutionArn)
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
	}
	if waitOutput.Datail != nil {
		log.Printf("[info] execution detail:\n%s", jsonutil.MarshalJSONString(waitOutput.Datail))
	}
	if waitOutput.Failed {
		return errors.New("state machine execution failed")
	}
	log.Printf("[info] execution success")
	if opt.Stdout != nil && len(waitOutput.Output) > 0 {
		io.WriteString(opt.Stdout, waitOutput.Output)
		io.WriteString(opt.Stdout, "\n")
	}
	return nil
}
