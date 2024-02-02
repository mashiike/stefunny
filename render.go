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
	def, err := app.cfg.LoadDefinition()
	if err != nil {
		return err
	}
	switch strings.ToLower(opt.Format) {
	case "dot":
		log.Println("[warn] dot format is deprecated (since v0.5.0)")
		stateMachine, err := asl.Parse(def)
		if err != nil {
			return err
		}
		bs, err := stateMachine.MarshalDOT(app.cfg.StateMachine.Name)
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
