package stefunny

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"

	"github.com/mashiike/stefunny/internal/asl"
)

type RenderOption struct {
	Writer io.Writer `kong:"-" json:"-"`
	Format string    `name:"format" help:"output format" default:"json" enum:"json,yaml,dot" json:"format,omitempty"`
}

func (app *App) Render(_ context.Context, opt RenderOption) error {
	if app.cfg.StateMachine == nil {
		return errors.New("state machine not found")
	}
	if app.cfg.StateMachine.Value.Definition == nil {
		return errors.New("state machine definition not found")
	}
	def := *app.cfg.StateMachine.Value.Definition
	switch strings.ToLower(opt.Format) {
	case "dot":
		log.Println("[warn] dot format is deprecated (since v0.5.0)")
		stateMachine, err := asl.Parse(def)
		if err != nil {
			return err
		}
		bs, err := stateMachine.MarshalDOT(app.cfg.StateMachineName())
		if err != nil {
			return err
		}
		_, err = opt.Writer.Write(bs)
		return err
	case "", "json":
		_, err := io.WriteString(opt.Writer, def)
		return err
	case "yaml":
		log.Println("[warn] yaml format is deprecated (since v0.5.0)")
		bs, err := JSON2YAML([]byte(def))
		if err != nil {
			return err
		}
		_, err = opt.Writer.Write(bs)
		return err
	}
	return errors.New("unknown format")
}
