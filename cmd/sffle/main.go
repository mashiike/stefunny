package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"

	"github.com/fatih/color"
	"github.com/fujiwara/logutils"
	"github.com/mashiike/sffle"
	"github.com/urfave/cli/v2"
)

var (
	Version = "current"
)

var filter = &logutils.LevelFilter{
	Levels:   []logutils.LogLevel{"debug", "info", "notice", "warn", "error"},
	MinLevel: "info",
	ModifierFuncs: []logutils.ModifierFunc{
		nil,
		logutils.Color(color.FgWhite),
		logutils.Color(color.FgHiBlue),
		logutils.Color(color.FgYellow),
		logutils.Color(color.FgRed, color.Bold),
	},
	Writer: os.Stderr,
}

func main() {
	app := &cli.App{
		Name:  "sffle",
		Usage: "A command line tool for deployment StepFunctions",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				DefaultText: "config.yaml",
				Usage:       "Load configuration from `FILE`",
				EnvVars:     []string{"SFFLE_CONFIG"},
			},
			&cli.StringFlag{
				Name:        "log-level",
				DefaultText: "info",
				Usage:       "Set log level (debug, info, notice, warn, error)",
				EnvVars:     []string{"SFFLE_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "deploy",
				Usage:  "deploy StepFunctions StateMachine.",
				Action: deply,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
				},
			},
			{
				Name:  "version",
				Usage: "show version info.",
				Action: func(c *cli.Context) error {
					setLogLevel(c)
					log.Printf("[info] sffle version     : %s", Version)
					log.Printf("[info] go runtime version: %s", runtime.Version())
					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	if err := app.Run(os.Args); err != nil {
		log.Printf("[error] %s", err)
	}
}

func setLogLevel(c *cli.Context) {
	level := c.String("log-level")
	if level != "" {
		filter.MinLevel = logutils.LogLevel(level)
	}
	log.SetOutput(filter)
	log.Println("[debug] Setting log level to", level)
}

func createApp(c *cli.Context) (*sffle.App, error) {
	cfg := sffle.NewDefaultConfig()
	if err := cfg.Load(c.String("config")); err != nil {
		return nil, err
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		os.Exit(1)
	}
	return sffle.New(context.Background(), cfg)
}

func deply(c *cli.Context) error {
	setLogLevel(c)
	app, err := createApp(c)
	if err != nil {
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)
	defer cancel()

	return app.Deploy(
		ctx,
		sffle.DeployOption{
			DryRun: c.Bool("dry-run"),
		},
	)
}
