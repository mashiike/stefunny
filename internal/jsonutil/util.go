package jsonutil

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/Cside/jsondiff"
	"github.com/fatih/color"
)

func JSONDiffString(j1, j2 string) string {
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

func marshalJSON(s interface{}) (*bytes.Buffer, error) {
	bs, err := buildJSON(s)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	json.Indent(&buf, bs, "", "  ")
	buf.WriteString("\n")
	return &buf, nil
}

func MarshalJSONString(s interface{}) string {
	b, _ := marshalJSON(s)
	return b.String()
}

func buildJSON(s interface{}) ([]byte, error) {
	bs, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var v map[string]interface{}
	if err := json.Unmarshal(bs, &v); err != nil {
		return nil, err
	}
	return json.Marshal(deleteNilFromMap(v))
}

func deleteNilFromMap(v map[string]interface{}) map[string]interface{} {
	for key, value := range v {
		if value == nil {
			delete(v, key)
			continue
		}
		if m, ok := value.(map[string]interface{}); ok {
			v[key] = deleteNilFromMap(m)
			continue
		}
		s, ok := value.([]interface{})
		if !ok {
			continue
		}
		replaceSlice := make([]interface{}, 0, len(s))
		for _, item := range s {
			if item == nil {
				continue
			}
			if item, ok := item.(map[string]interface{}); ok {
				replaceSlice = append(replaceSlice, deleteNilFromMap(item))
				continue
			}
			replaceSlice = append(replaceSlice, item)
		}
		v[key] = replaceSlice
	}
	return v
}
