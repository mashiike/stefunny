package jsonutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"strings"

	"github.com/Cside/jsondiff"
	"github.com/fatih/color"
	"gopkg.in/yaml.v3"
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
	if err := json.Indent(&buf, bs, "", "  "); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString("\n"); err != nil {
		return nil, err
	}
	return &buf, nil
}

func MarshalJSONString(s interface{}) string {
	b, err := marshalJSON(s)
	if err != nil {
		log.Println("[warn] failed to marshal json", err)
		return ""
	}
	return b.String()
}

func buildJSON(s interface{}) ([]byte, error) {
	bs, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var v interface{}
	if err := json.Unmarshal(bs, &v); err != nil {
		return nil, err
	}
	if v, ok := v.(map[string]interface{}); ok {
		return json.Marshal(deleteNilFromMap(v))
	}
	if vs, ok := v.([]interface{}); ok {
		for i := 0; i < len(vs); i++ {
			if v, ok := vs[i].(map[string]interface{}); ok {
				vs[i] = deleteNilFromMap(v)
			}
		}
		return json.Marshal(vs)
	}
	return json.Marshal(v)
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

func Yaml2Json(data []byte) ([]byte, error) {
	var temp map[string]interface{}
	if err := yaml.Unmarshal(data, &temp); err != nil {
		return nil, err
	}
	m, err := convertKeyString(temp)
	if err != nil {
		return nil, err
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func JSON2YAML(data []byte) ([]byte, error) {
	var temp map[string]interface{}
	if err := json.Unmarshal(data, &temp); err != nil {
		return nil, err
	}
	return yaml.Marshal(temp)
}

func convertKeyString(v interface{}) (interface{}, error) {
	switch cv := v.(type) {
	case map[string]interface{}:
		ret := make(map[string]interface{}, len(cv))
		for key, value := range cv {
			var err error
			ret[key], err = convertKeyString(value)
			if err != nil {
				return nil, err
			}
		}
		return ret, nil
	case map[interface{}]interface{}:
		ret := make(map[string]interface{}, len(cv))
		for key, value := range cv {
			skey, ok := key.(string)
			if !ok {
				return errors.New("can not convert key string"), nil
			}
			var err error
			ret[skey], err = convertKeyString(value)
			if err != nil {
				return nil, err
			}
		}
		return ret, nil
	}
	return v, nil
}
