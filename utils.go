package stefunny

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/Songmu/prompter"
	"github.com/aws/aws-sdk-go-v2/aws/arn"
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

func ptr[T any](v T) *T {
	return &v
}

func extructVersion(versionARN string) (int, error) {
	arnObj, err := arn.Parse(versionARN)
	if err != nil {
		return 0, fmt.Errorf("parse arn failed: %w", err)
	}
	parts := strings.Split(arnObj.Resource, ":")
	if parts[0] != "stateMachine" {
		return 0, fmt.Errorf("`%s` is not state machine version arn", versionARN)
	}
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid arn format: %s", versionARN)
	}
	version, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("parse version number failed: %w", err)
	}
	return version, nil
}
