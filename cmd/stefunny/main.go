package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"syscall"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/logger"
	"github.com/urfave/cli/v2"
)

var (
	Version = "current"
	app     *stefunny.App
)

func main() {
	cliApp := &cli.App{
		Name:  "stefunny",
		Usage: "A command line tool for deployment StepFunctions",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				DefaultText: "config.yaml",
				Usage:       "Load configuration from `FILE`",
				EnvVars:     []string{"STEFUNNY_CONFIG"},
			},
			&cli.StringFlag{
				Name:        "log-level",
				DefaultText: "info",
				Usage:       "Set log level (debug, info, notice, warn, error)",
				EnvVars:     []string{"STEFUNNY_LOG_LEVEL"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "create",
				Usage: "create StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					return app.Create(c.Context, stefunny.DeployOption{
						DryRun: c.Bool("dry-run"),
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
				},
			},
			{
				Name:  "delete",
				Usage: "delete StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					return app.Delete(c.Context, stefunny.DeleteOption{
						DryRun: c.Bool("dry-run"),
						Force:  c.Bool("force"),
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
					&cli.BoolFlag{
						Name:  "force",
						Usage: "delete without confirmation",
					},
				},
			},
			{
				Name:  "deploy",
				Usage: "deploy StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					return app.Deploy(c.Context, stefunny.DeployOption{
						DryRun: c.Bool("dry-run"),
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
				},
			},
			{
				Name:  "render",
				Usage: "render state machie defienion(the Amazon States Language) as a dot file",
				Action: func(c *cli.Context) error {
					args := c.Args()
					opt := stefunny.RenderOption{
						Writer: os.Stdin,
					}
					if args.Len() > 0 {
						path := args.First()
						log.Printf("[notice] output to %s", path)
						fp, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
						if err != nil {
							return err
						}
						defer fp.Close()
						opt.Writer = fp
					}
					return app.Render(c.Context, opt)
				},
			},
			{
				Name:  "version",
				Usage: "show version info.",
				Action: func(c *cli.Context) error {
					log.Printf("[info] stefunny version     : %s", Version)
					log.Printf("[info] go runtime version: %s", runtime.Version())
					return nil
				},
			},
		},
	}

	sort.Sort(cli.FlagsByName(cliApp.Flags))
	sort.Sort(cli.CommandsByName(cliApp.Commands))
	cliApp.Before = func(c *cli.Context) error {
		logger.Setup(os.Stderr, c.String("log-level"))

		cfg := stefunny.NewDefaultConfig()
		if err := cfg.Load(c.String("config")); err != nil {
			return err
		}
		if err := cfg.ValidateVersion(Version); err != nil {
			return err
		}
		var err error
		app, err = stefunny.New(c.Context, cfg)
		return err
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, os.Interrupt)
	defer cancel()

	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		log.Printf("[error] %s", err)
	}
}
