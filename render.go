package stefunny

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
)

type RenderOption struct {
	Writer  io.Writer `kong:"-" json:"-"`
	Targets []string  `arg:"" help:"target to render (config, definition, def)" enum:"config,definition,def" json:"targets,omitempty"`
	Format  string    `name:"format" help:"output format(json, jsonnet, yaml)" default:"" enum:",json,jsonnet,yaml" json:"format,omitempty"`
}

func (app *App) Render(ctx context.Context, opt RenderOption) error {
	out := bufio.NewWriter(opt.Writer)
	defer out.Flush()
	renderer := NewRenderer(app.cfg)

	for _, target := range opt.Targets {
		switch target {
		case "config":
			format := opt.Format
			if format == "" {
				format = "yaml"
			}
			if err := renderer.RenderConfig(ctx, out, format, false); err != nil {
				return err
			}
		case "definition", "def":
			format := opt.Format
			if format == "" {
				format = "json"
			}
			if err := renderer.RenderStateMachine(ctx, out, format, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown target: %s", target)
		}
		out.WriteRune('\n')
	}
	return nil
}

type Renderer struct {
	cfg                    *Config
	cachedTFstateResources *OrderdMap[string, string]
}

func NewRenderer(cfg *Config) *Renderer {
	return &Renderer{
		cfg: cfg,
	}
}

func (r *Renderer) CreateConfigFile(ctx context.Context, path string, template bool) error {
	fmt, err := r.detectFormat(path)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.RenderConfig(ctx, f, fmt, template)
}

func (r *Renderer) RenderConfig(ctx context.Context, w io.Writer, format string, template bool) error {
	def := r.cfg.StateMachineDefinition()
	r.cfg.StateMachine.SetDefinition(r.cfg.StateMachine.DefinitionPath)
	defer func() {
		r.cfg.StateMachine.SetDefinition(def)
	}()
	var v any = r.cfg
	if template {
		var err error
		v, err = r.templateize(ctx, r.cfg)
		if err != nil {
			return fmt.Errorf("failed to templateize: %w", err)
		}
	}
	if err := r.render(w, format, v); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	return nil
}

func (r *Renderer) CreateDefinitionFile(ctx context.Context, path string, template bool) error {
	fmt, err := r.detectFormat(path)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.RenderStateMachine(ctx, f, fmt, template)
}

func (r *Renderer) RenderStateMachine(ctx context.Context, w io.Writer, format string, template bool) error {
	def := json.RawMessage(r.cfg.StateMachineDefinition())
	var v any = def
	if template {
		var err error
		v, err = r.templateize(ctx, def)
		if err != nil {
			return fmt.Errorf("failed to templateize: %w", err)
		}
	}
	if err := r.render(w, format, v); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	return nil
}

func (r *Renderer) detectFormat(path string) (string, error) {
	ext := filepath.Ext(path)
	switch strings.ToLower(ext) {
	case jsonExt:
		return "json", nil
	case jsonnetExt:
		return "jsonnet", nil
	case yamlExt, ymlExt:
		return "yaml", nil
	default:
		return "", fmt.Errorf("unknown file extension: %s", ext)
	}
}

func (r *Renderer) render(w io.Writer, format string, v any) error {
	switch f := strings.ToLower(format); f {
	case "json", "jsonnet":
		buf, err := marshalJSON(v)
		if err != nil {
			return err
		}
		if f == "json" {
			_, err = w.Write(buf.Bytes())
			return err
		}
		bs, err := JSON2Jsonnet(r.cfg.ConfigDir, buf.Bytes())
		if err != nil {
			return err
		}
		_, err = w.Write(bs)
		return err
	case "yaml":
		enc := yaml.NewEncoder(w, yaml.UseJSONMarshaler())
		if err := enc.Encode(v); err != nil {
			return err
		}
		return enc.Close()
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func (r *Renderer) templateize(ctx context.Context, v any) (any, error) {
	bs, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal: %w", err)
	}
	var data any
	if err := json.Unmarshal(bs, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal: %w", err)
	}
	for _, tfstateCfg := range r.cfg.TFState {
		data, err = r.templateizeTFState(ctx, data, tfstateCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to templateize for tfstate `%s`: %w", tfstateCfg.Location, err)
		}
	}
	if r.cfg.MustEnvs.Len() > 0 {
		data, err = r.templateizeMustEnvs(data, r.cfg.MustEnvs)
		if err != nil {
			return nil, fmt.Errorf("failed to templateize for must_env: %w", err)
		}
	}
	if r.cfg.Envs.Len() > 0 {
		data, err = r.templateizeEnvs(data, r.cfg.Envs)
		if err != nil {
			return nil, fmt.Errorf("faield to templateize for env: %w", err)
		}
	}
	return data, nil
}

func (r *Renderer) templateizeTFState(ctx context.Context, data any, cfg *TFStateConfig) (any, error) {
	resources := r.cachedTFstateResources
	if resources == nil {
		var err error
		resources, err = ListResourcesFromTFState(ctx, cfg.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to list resources from tfstate `%s`: %w", cfg.Location, err)
		}
		r.cachedTFstateResources = resources
	}
	keys := resources.Keys()
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		value, ok := resources.Get(key)
		if !ok {
			continue
		}
		data = walkStringReplaceAll(data, value, fmt.Sprintf("{{ tfstate `%s` }}", key))
	}
	return data, nil
}

func (r *Renderer) templateizeMustEnvs(data any, envs *OrderdMap[string, string]) (any, error) {
	keys := envs.Keys()
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		value, ok := envs.Get(key)
		if !ok {
			continue
		}
		data = walkStringReplaceAll(data, value, fmt.Sprintf("{{ must_env `%s` }}", key))
	}
	return data, nil
}

func (r *Renderer) templateizeEnvs(data any, envs *OrderdMap[string, string]) (any, error) {
	keys := envs.Keys()
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		value, ok := envs.Get(key)
		if !ok {
			continue
		}
		if value == "" {
			continue
		}
		args := strings.Split(key, ",")
		fields := make([]string, len(args))
		for i, arg := range args {
			fields[i] = "`" + arg + "`"
		}
		data = walkStringReplaceAll(data, value, fmt.Sprintf("{{ env %s }}", strings.Join(fields, " ")))
	}
	return data, nil
}

func walkStringReplaceAll(v any, from, to string) any {
	switch x := v.(type) {
	case string:
		return strings.ReplaceAll(x, from, to)
	case map[string]interface{}:
		for k, vv := range x {
			x[k] = walkStringReplaceAll(vv, from, to)
		}
		return x
	case []interface{}:
		for i, vv := range x {
			x[i] = walkStringReplaceAll(vv, from, to)
		}
		return x
	default:
		return v
	}
}
