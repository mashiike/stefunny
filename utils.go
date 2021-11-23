package stefunny

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Songmu/prompter"
	"github.com/fatih/color"
)

func colorRestString(str string) string {
	var buf bytes.Buffer
	c := color.New(color.Reset)
	c.Fprint(&buf, str)
	return buf.String()
}

func prompt(ctx context.Context, msg string, defaultInput string) (string, error) {
	var input string
	ch := make(chan struct{})
	go func() {
		input = prompter.Prompt(msg, defaultInput)
		close(ch)
	}()
	select {
	case <-ctx.Done():
		fmt.Print("\n")
		return defaultInput, ctx.Err()
	case <-ch:
		return input, nil
	}
}

func coalesceString(str *string, d string) string {
	if str == nil {
		return d
	}
	if *str == "" {
		return d
	}
	return *str
}
