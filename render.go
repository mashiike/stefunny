package stefunny

import (
	"bufio"
	"bytes"
	"context"
	"errors"
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

func (app *App) Render(_ context.Context, opt RenderOption) error {
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
			if err := renderer.RenderConfig(out, format, false); err != nil {
				return err
			}
		case "definition", "def":
			format := opt.Format
			if format == "" {
				format = "json"
			}
			if err := renderer.RenderStateMachine(out, format, false); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown target: %s", target)
		}
	}
	return nil
}

type Renderer struct {
	cfg *Config
}

func NewRenderer(cfg *Config) *Renderer {
	return &Renderer{
		cfg: cfg,
	}
}

func (r *Renderer) CreateConfigFile(path string, template bool) error {
	fmt, err := r.detectFormat(path)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.RenderConfig(f, fmt, template)
}

func (r *Renderer) RenderConfig(w io.Writer, format string, template bool) error {
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
	var buf bytes.Buffer
	if err := r.render(&buf, format, r.cfg); err != nil {
		return fmt.Errorf("failed to render: %w", err)
	}
	if err := r.templateize(w, &buf); err != nil {
		return fmt.Errorf("failed to templateize: %w", err)
	}
	return r.templateize(w, &buf)
}

func (r *Renderer) CreateDefinitionFile(path string, template bool) error {
	fmt, err := r.detectFormat(path)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return r.RenderStateMachine(f, fmt, template)
}

func (r *Renderer) RenderStateMachine(w io.Writer, format string, template bool) error {
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
	if err := r.templateize(w, &buf); err != nil {
		return fmt.Errorf("failed to templateize: %w", err)
	}
	return r.templateize(w, &buf)
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

func (r *Renderer) templateize(writer io.Writer, reader io.Reader) error {
	return errors.New("not implemented yet")
}
