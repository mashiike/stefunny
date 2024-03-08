package stefunny

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func coalesce[T any](ptrs ...*T) T {
	for _, ptr := range ptrs {
		if ptr != nil {
			return *ptr
		}
	}
	var zero T
	return zero
}

func ptr[T any](v T) *T {
	return &v
}

func extructVersion(versionArn string) (int, error) {
	arnObj, err := arn.Parse(versionArn)
	if err != nil {
		return 0, fmt.Errorf("parse arn failed: %w", err)
	}
	parts := strings.Split(arnObj.Resource, ":")
	if parts[0] != "stateMachine" {
		return 0, fmt.Errorf("`%s` is not state machine version arn", versionArn)
	}
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid arn format: %s", versionArn)
	}
	if len(parts) == 2 {
		return 0, nil
	}
	version, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, fmt.Errorf("parse version number failed: %w", err)
	}
	return version, nil
}

func addQualifierToArn(arnStr string, name string) string {
	if name == "" || name == "0" {
		return arnStr
	}
	return fmt.Sprintf("%s:%s", arnStr, name)
}

func removeQualifierFromArn(arnStr string) string {
	arnObj, err := arn.Parse(arnStr)
	if err != nil {
		return arnStr
	}
	parts := strings.Split(arnObj.Resource, ":")
	if parts[0] != "stateMachine" {
		return arnStr
	}
	if len(parts) <= 2 {
		// case state machine arn
		return arnStr
	}
	// case qualified state machine arn, delete version or alias.
	// e.g. arn:aws:states:us-west-2:123456789012:stateMachine:HelloWorld-StateMachine:1
	arnObj.Resource = strings.Join(parts[:2], ":")
	return arnObj.String()
}

func unique[T comparable](slice []T) []T {
	result := make([]T, 0)
	seen := make(map[T]struct{})
	for _, item := range slice {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return false
	}
	return v.IsNil()
}

type change[T any] struct {
	Before T
	After  T
}

type sliceDiffResult[T any] struct {
	Add    []T
	Delete []T
	Change []change[T]
}

// diff for this -> other
func sliceDiff[T any](this, other []T, fetchKey func(T) string) sliceDiffResult[T] {
	result := sliceDiffResult[T]{}
	thisMap := make(map[string]T)
	for _, item := range this {
		thisMap[fetchKey(item)] = item
	}
	otherMap := make(map[string]T)
	for _, item := range other {
		otherMap[fetchKey(item)] = item
	}
	for key, item := range thisMap {
		if _, ok := otherMap[key]; !ok {
			result.Delete = append(result.Delete, item)
			continue
		}
		result.Change = append(result.Change, change[T]{Before: item, After: otherMap[key]})
	}
	for key, item := range otherMap {
		if _, ok := thisMap[key]; !ok {
			result.Add = append(result.Add, item)
		}
	}
	return result
}

// create file with mkdir
func createFileWithMkdir(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("mkdir failed: %w", err)
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create file failed: %w", err)
	}
	return file, nil
}
