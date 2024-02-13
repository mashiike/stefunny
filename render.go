package stefunny

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
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
	if !template {
		if err := r.render(w, format, r.cfg); err != nil {
			return fmt.Errorf("failed to render: %w", err)
		}
		return nil
	}
	buf := new(bytes.Buffer)
	if err := r.render(buf, format, r.cfg); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	if err := r.templateize(ctx, w, buf); err != nil {
		return fmt.Errorf("failed to templateize: %w", err)
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
	def := JSONRawMessage(r.cfg.StateMachineDefinition())
	if !template {
		if err := r.render(w, format, def); err != nil {
			return fmt.Errorf("failed to render: %w", err)
		}
		return nil
	}
	var buf bytes.Buffer
	if err := r.render(&buf, format, def); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	if err := r.templateize(ctx, w, &buf); err != nil {
		return fmt.Errorf("failed to templateize: %w", err)
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
		enc := yaml.NewEncoder(w)
		if err := enc.Encode(v); err != nil {
			return err
		}
		return enc.Close()
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

func (r *Renderer) templateize(ctx context.Context, writer io.Writer, reader io.Reader) error {
	bs, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	for _, tfstateCfg := range r.cfg.TFState {
		bs, err = r.templateizeTFState(ctx, bs, tfstateCfg)
		if err != nil {
			return fmt.Errorf("failed to templateize for tfstate `%s`: %w", tfstateCfg.Location, err)
		}
	}
	if r.cfg.MustEnvs.Len() > 0 {
		bs, err = r.templateizeMustEnvs(bs, r.cfg.MustEnvs)
		if err != nil {
			return fmt.Errorf("failed to templateize for must_env: %w", err)
		}
	}
	if r.cfg.Envs.Len() > 0 {
		bs, err = r.templateizeEnvs(bs, r.cfg.Envs)
		if err != nil {
			return fmt.Errorf("faield to templatize for env: %w", err)
		}
	}
	_, err = writer.Write(bs)
	if err != nil {
		return fmt.Errorf("failed to write: %w", err)
	}
	io.WriteString(writer, "\n")

	return nil
}

func (r *Renderer) templateizeTFState(ctx context.Context, bs []byte, cfg *TFStateConfig) ([]byte, error) {
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
		bs = bytes.ReplaceAll(bs, []byte(value), []byte(fmt.Sprintf("{{ %stfstate `%s` }}", cfg.FuncPrefix, key)))
	}
	return bs, nil
}

func (r *Renderer) templateizeMustEnvs(bs []byte, envs *OrderdMap[string, string]) ([]byte, error) {
	keys := envs.Keys()
	for i := len(keys) - 1; i >= 0; i-- {
		key := keys[i]
		value, ok := envs.Get(key)
		if !ok {
			continue
		}
		bs = bytes.ReplaceAll(bs, []byte(value), []byte(fmt.Sprintf("{{ must_env `%s` }}", key)))
	}
	return bs, nil
}

func (r *Renderer) templateizeEnvs(bs []byte, envs *OrderdMap[string, string]) ([]byte, error) {
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
		bs = bytes.ReplaceAll(bs, []byte(value), []byte(fmt.Sprintf("{{ env %s }}", strings.Join(fields, " "))))
	}
	return bs, nil
}
