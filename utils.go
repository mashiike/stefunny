package stefunny

import (
	"bytes"
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

func colorRestString(str string) string {
	var buf bytes.Buffer
	c := color.New(color.Reset)
	c.Fprint(&buf, str)
	return buf.String()
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

func marshalJSONString(s interface{}) string {
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
		}
		if m, ok := value.(map[string]interface{}); ok {
			v[key] = deleteNilFromMap(m)
		}
	}
	return v
}
