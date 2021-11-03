package sffle

import (
	"encoding/json"
	"strings"

	"github.com/Cside/jsondiff"
	"github.com/fatih/color"
)

func jsonDiffString(j1, j2 string) string {
	diff := jsondiff.Diff([]byte(j1), []byte(j2))
	var builder strings.Builder
	c := color.New(color.Reset)
	if diff == "" {
		c.Fprint(&builder, j1, "\n")
		return builder.String()
	}
	diffLines := strings.Split(diff, "\n")
	for _, str := range diffLines {
		trimStr := strings.TrimSpace(str)
		if strings.HasPrefix(trimStr, "+") {
			builder.WriteString(color.GreenString(str) + "\n")
			continue
		}
		if strings.HasPrefix(trimStr, "-") {
			builder.WriteString(color.RedString(str) + "\n")
			continue
		}
		c.Fprint(&builder, str, "\n")
	}
	return builder.String()
}

func definitionToMap(def string) (map[string]interface{}, error) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(def), &m); err != nil {
		return nil, err
	}
	return m, nil
}
