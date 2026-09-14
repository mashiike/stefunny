package stefunny

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-jsonnet/formatter"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/itchyny/gojq"
	"github.com/kylelemons/godebug/diff"
	"github.com/serenize/snaker"
)

func JSON2Jsonnet(filename string, data []byte) ([]byte, error) {
	formattted, err := formatter.Format(filename, string(data), formatter.DefaultOptions())
	if err != nil {
		return data, err
	}
	return []byte(formattted), nil
}

func toDiffString(s1 string) string {
	if strings.EqualFold(s1, "null") || strings.EqualFold(s1, "null\n") {
		return ""
	}
	return s1
}

type jsonDiffParams struct {
	unified bool
	fromURI string
	toURI   string
	ignore  string
}

type JSONDiffOption func(*jsonDiffParams)

func JSONDiffFromURI(uri string) JSONDiffOption {
	return func(p *jsonDiffParams) {
		p.fromURI = uri
	}
}

func JSONDiffToURI(uri string) JSONDiffOption {
	return func(p *jsonDiffParams) {
		p.toURI = uri
	}
}

func JSONDiffUnified(b bool) JSONDiffOption {
	return func(p *jsonDiffParams) {
		p.unified = b
	}
}

// JSONDiffIgnore sets a jq query whose matched paths are removed from both
// sides before they are compared. query is wrapped as del(<query>); an
// empty query disables filtering.
func JSONDiffIgnore(query string) JSONDiffOption {
	return func(p *jsonDiffParams) {
		p.ignore = query
	}
}

// JSONDiffString renders a color-highlighted diff between fromStr and
// toStr, applying opts (see JSONDiffFromURI, JSONDiffToURI, JSONDiffUnified,
// JSONDiffIgnore). Returns an error if a JSONDiffIgnore query is invalid or
// fails to apply.
func JSONDiffString(fromStr, toStr string, opts ...JSONDiffOption) (string, error) {
	var b strings.Builder
	str, err := jsonDiffString(fromStr, toStr, opts...)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(str, "\n") {
		if strings.HasPrefix(line, "-") {
			b.WriteString(color.RedString(line) + "\n")
		} else if strings.HasPrefix(line, "+") {
			b.WriteString(color.GreenString(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}
	return b.String(), nil
}

func jsonDiffString(fromStr, toStr string, opts ...JSONDiffOption) (string, error) {
	var params jsonDiffParams
	for _, opt := range opts {
		opt(&params)
	}
	fromStr = toDiffString(fromStr)
	if params.ignore != "" {
		var err error
		if fromStr, err = applyJQIgnore(fromStr, params.ignore); err != nil {
			return "", fmt.Errorf("apply ignore query to %s: %w", params.fromURI, err)
		}
	}
	if fromStr != "" {
		var fromBuf bytes.Buffer
		if err := json.Indent(&fromBuf, []byte(fromStr), "", "  "); err != nil {
			log.Println("[warn] failed to indent json", err)
		}
		fromStr = fromBuf.String()
	}
	toStr = toDiffString(toStr)
	if params.ignore != "" {
		var err error
		if toStr, err = applyJQIgnore(toStr, params.ignore); err != nil {
			return "", fmt.Errorf("apply ignore query to %s: %w", params.toURI, err)
		}
	}
	if toStr != "" {
		var toBuf bytes.Buffer
		if err := json.Indent(&toBuf, []byte(toStr), "", "  "); err != nil {
			log.Println("[warn] failed to indent json", err)
		}
		toStr = toBuf.String()
	}
	if strings.EqualFold(fromStr, "null") || strings.EqualFold(fromStr, "null\n") {
		fromStr = ""
	} else {
		if !strings.HasSuffix(fromStr, "\n") {
			fromStr += "\n"
		}
	}

	if strings.EqualFold(toStr, "null") || strings.EqualFold(toStr, "null\n") {
		toStr = ""
	} else {
		if !strings.HasSuffix(toStr, "\n") {
			toStr += "\n"
		}
	}

	if params.unified {
		edits := myers.ComputeEdits(span.URIFromPath(params.fromURI), fromStr, toStr)
		return fmt.Sprint(gotextdiff.ToUnified(params.fromURI, params.toURI, fromStr, edits)), nil
	}

	ds := diff.Diff(fromStr, toStr)
	if ds == "" {
		return ds, nil
	}
	return fmt.Sprintf("--- %s\n+++ %s\n%s", params.fromURI, params.toURI, ds), nil
}

func applyJQIgnore(s, query string) (string, error) {
	if s == "" {
		return s, nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", fmt.Errorf("unmarshal json: %w", err)
	}
	filtered, err := deleteByJQ(v, query)
	if err != nil {
		return "", err
	}
	bs, err := json.Marshal(filtered)
	if err != nil {
		return "", fmt.Errorf("marshal json: %w", err)
	}
	return string(bs), nil
}

func deleteByJQ(v interface{}, query string) (interface{}, error) {
	pathQuery, err := gojq.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("parse jq query %q: %w", query, err)
	}
	q := &gojq.Query{
		Term: &gojq.Term{
			Type: gojq.TermTypeFunc,
			Func: &gojq.Func{
				Name: "del",
				Args: []*gojq.Query{pathQuery},
			},
		},
	}
	iter := q.Run(v)
	result := v
	for {
		next, ok := iter.Next()
		if !ok {
			break
		}
		if err, ok := next.(error); ok {
			return nil, fmt.Errorf("run jq query %q: %w", query, err)
		}
		result = next
	}
	return result, nil
}

func marshalJSON(s interface{}, overrides ...any) (*bytes.Buffer, error) {
	bs, err := buildJSON(s, overrides...)
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

func MarshalJSONString(s interface{}, overrides ...any) string {
	b, err := marshalJSON(s, overrides...)
	if err != nil {
		log.Println("[warn] failed to marshal json", err)
		return ""
	}
	return b.String()
}

func buildJSON(s interface{}, overrides ...any) ([]byte, error) {
	bs, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	var v interface{}
	if err := json.Unmarshal(bs, &v); err != nil {
		return nil, err
	}
	o := make([]interface{}, 0, len(overrides))
	for _, override := range overrides {
		bs, err := json.Marshal(override)
		if err != nil {
			return nil, err
		}
		var ov interface{}
		if err := json.Unmarshal(bs, &ov); err != nil {
			return nil, err
		}
		o = append(o, ov)
	}
	if v, ok := v.(map[string]interface{}); ok {
		if len(overrides) > 0 {
			for _, override := range o {
				if override, ok := override.(map[string]interface{}); ok {
					for key, value := range override {
						v[key] = value
					}
				}
			}
		}
		return json.Marshal(deleteNilFromMap(v))
	}
	if vs, ok := v.([]interface{}); ok {
		if len(overrides) > 0 {
			for _, override := range o {
				if override, ok := override.([]interface{}); ok {
					vs = append(vs, override...)
				}
			}
		}
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
		if isNil(value) {
			delete(v, key)
			continue
		}
		if str, ok := value.(string); ok {
			if str == "" {
				delete(v, key)
			}
			continue
		}
		if b, ok := value.(bool); ok {
			if !b {
				delete(v, key)
			}
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
		if len(replaceSlice) == 0 {
			delete(v, key)
			continue
		}
		v[key] = replaceSlice
	}
	return v
}

// KeysToSnakeCase converts the keys of the given object to snake case.
// The given object is expected struct, json struct key is CamelCase.
type KeysToSnakeCase[T any] struct {
	Value  T
	Strict bool `yaml:"-"`
}

func NewKeysToSnakeCase[T any](v T) KeysToSnakeCase[T] {
	return KeysToSnakeCase[T]{Value: v}
}

func SnakeToCamel(s string) string {
	str := snaker.SnakeToCamel(s)
	str = strings.ReplaceAll(str, "Cloudwatch", "CloudWatch")
	return str
}

func CamelToSnake(s string) string {
	str := snaker.CamelToSnake(s)
	str = strings.ReplaceAll(str, "cloud_watch", "cloudwatch")
	return str
}

func (k *KeysToSnakeCase[T]) UnmarshalJSON(bs []byte) error {
	var data map[string]any
	if err := json.Unmarshal(bs, &data); err != nil {
		return err
	}
	if err := walkMap(data, SnakeToCamel); err != nil {
		return err
	}
	bs, err := json.Marshal(data)
	if err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(bs))
	if k.Strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(&k.Value); err != nil {
		return err
	}
	return nil
}

func (k KeysToSnakeCase[T]) MarshalYAML() (interface{}, error) {
	bs, err := buildJSON(k.Value)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, err
	}
	if err := walkMap(data, CamelToSnake); err != nil {
		return nil, err
	}
	return data, nil
}

func (k KeysToSnakeCase[T]) MarshalJSON() ([]byte, error) {
	bs, err := buildJSON(k.Value)
	if err != nil {
		return nil, err
	}
	var data map[string]any
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, err
	}
	if err := walkMap(data, CamelToSnake); err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

func walkMap(data map[string]any, keyModifier func(string) string) error {
	for k, v := range data {
		delete(data, k)
		newKey := keyModifier(k)
		if v == nil {
			continue
		}
		data[newKey] = v
		switch v := v.(type) {
		case map[string]any:
			if err := walkMap(v, keyModifier); err != nil {
				return err
			}
		case []any:
			if err := walkSlilce(v, keyModifier); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func walkSlilce(data []any, keyModifier func(string) string) error {
	for i := 0; i < len(data); i++ {
		switch v := data[i].(type) {
		case map[string]any:
			if err := walkMap(v, keyModifier); err != nil {
				return err
			}
		case []any:
			if err := walkSlilce(v, keyModifier); err != nil {
				return err
			}
		default:
			continue
		}
	}
	return nil
}

func jsonEscape(v string) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	bs := bytes.TrimSpace(buf.Bytes())
	return string(bs[1 : len(bs)-1]), nil
}
