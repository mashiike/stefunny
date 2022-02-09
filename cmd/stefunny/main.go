package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strings"
	"syscall"

	"github.com/mashiike/stefunny"
	"github.com/mashiike/stefunny/internal/logger"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh/terminal"
)

var (
	Version       = "current"
	globalConfig  string
	globalTFState string
)

func main() {
	cliApp := &cli.App{
		Name:  "stefunny",
		Usage: "A command line tool for deployment StepFunctions and EventBridge",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "config",
				Aliases:     []string{"c"},
				DefaultText: "config.yaml",
				Usage:       "Load configuration from `FILE`",
				EnvVars:     []string{"STEFUNNY_CONFIG"},
				Destination: &globalConfig,
			},
			&cli.StringFlag{
				Name:        "log-level",
				DefaultText: "info",
				Usage:       "Set log level (debug, info, notice, warn, error)",
				EnvVars:     []string{"STEFUNNY_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:        "tfstate",
				DefaultText: "",
				Usage:       "URL to terraform.tfstate referenced in config",
				EnvVars:     []string{"STEFUNNY_TFSTATE"},
				Destination: &globalTFState,
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "init",
				Usage:     "Initialize stefunny from an existing StateMachine",
				UsageText: "stefunny init [options] --state-machine <state machine name>",
				Action: func(c *cli.Context) error {
					cfg := stefunny.NewDefaultConfig()
					cfg.AWSRegion = c.String("region")
					app, err := stefunny.New(c.Context, cfg)
					if err != nil {
						return err
					}
					return app.Init(c.Context, &stefunny.InitInput{
						Version:            Version,
						AWSRegion:          c.String("region"),
						ConfigPath:         c.String("config"),
						DefinitionFileName: c.String("definition"),
						StateMachineName:   c.String("state-machine"),
					})
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "region",
						EnvVars: []string{"AWS_REGION", "AWS_DEAULT_REGION"},
					},
					&cli.StringFlag{
						Name:    "config",
						Aliases: []string{"c"},
						Value:   "config.yaml",
						Usage:   "save configuration to `FILE`",
					},
					&cli.StringFlag{
						Name:    "definition",
						Aliases: []string{"d"},
						Value:   "definition.jsonnet",
						Usage:   "save definition to `FILE`",
					},
					&cli.StringFlag{
						Name:     "state-machine",
						Required: true,
						Aliases:  []string{"s"},
						Usage:    "existing state machine name",
					},
				},
			},
			{
				Name:  "create",
				Usage: "create StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
					return app.Create(c.Context, stefunny.DeployOption{
						DryRun: c.Bool("dry-run"),
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
				},
			},
			{
				Name:  "delete",
				Usage: "delete StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
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
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
				},
			},
			{
				Name:  "deploy",
				Usage: "deploy StepFunctions StateMachine and Event Bridge Rule.",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
					return app.Deploy(c.Context, stefunny.DeployOption{
						DryRun: c.Bool("dry-run"),
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
				},
			},
			{
				Name:  "schedule",
				Usage: "schedule Bridge Rule without deploy StepFunctions StateMachine.",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
					enabled := c.Bool("enabled")
					disabled := c.Bool("disabled")
					var setStatus *bool
					if enabled && disabled {
						return errors.New("both enabled and disabled are specified")
					}
					if enabled {
						setStatus = &enabled
					}
					if disabled {
						disabled = false
						setStatus = &disabled
					}
					return app.Deploy(c.Context, stefunny.DeployOption{
						DryRun:                 c.Bool("dry-run"),
						ScheduleEnabled:        setStatus,
						SkipDeployStateMachine: true,
					})
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "dry-run",
						Usage: "dry run",
					},
					&cli.BoolFlag{
						Name:  "enabled",
						Usage: "set schedule rule enabled",
					},
					&cli.BoolFlag{
						Name:  "disabled",
						Usage: "set schedule rule disabled",
					},
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
				},
			},
			{
				Name:  "render",
				Usage: "render state machie definition(the Amazon States Language) as a dot file",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
					args := c.Args()
					opt := stefunny.RenderOption{
						Writer: os.Stdin,
						Format: c.String("format"),
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
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
					&cli.StringFlag{
						Name:        "format",
						DefaultText: "dot",
						Usage:       "state machine render format (dot|json|yaml)",
					},
				},
			},
			{
				Name:  "execute",
				Usage: "execute state machine",
				Action: func(c *cli.Context) error {
					app, err := buildApp(c)
					if err != nil {
						return err
					}
					opt := stefunny.ExecuteOption{
						Stdin:         os.Stdin,
						Stdout:        os.Stdout,
						ExecutionName: c.String("execution-name"),
						Async:         c.Bool("async"),
					}
					switch {
					case c.Bool("stdin"):
						// nothing todo
					case c.String("input") != "":
						fp, err := os.Open(c.String("input"))
						if err != nil {
							return err
						}
						defer fp.Close()
						opt.Stdin = fp
					case terminal.IsTerminal(syscall.Stdin):
						defaultInput := `{"Comment": "Insert your JSON here"}`
						log.Println("[warn] no input is specified, so we'll use the default input in .")
						opt.Stdin = strings.NewReader(defaultInput)
					}
					return app.Execute(c.Context, opt)
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "config",
						Aliases:     []string{"c"},
						DefaultText: "config.yaml",
						Usage:       "Load configuration from `FILE`",
					},
					&cli.StringFlag{
						Name:        "tfstate",
						DefaultText: "",
						Usage:       "URL to terraform.tfstate referenced in config",
					},
					&cli.BoolFlag{
						Name:  "stdin",
						Usage: "input from stdin",
					},
					&cli.StringFlag{
						Name:        "input",
						DefaultText: "",
						Usage:       "input JSON filepath",
					},
					&cli.StringFlag{
						Name:        "execution-name",
						DefaultText: "",
						Usage:       "execution name: if not provide, generate uuid version 4 execution name",
					},
					&cli.BoolFlag{
						Name:  "async",
						Usage: "execution type async: no wait finish execution",
					},
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
	cliApp.Version = Version
	cliApp.Before = func(c *cli.Context) error {
		logger.Setup(os.Stderr, c.String("log-level"))
		return nil
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, os.Interrupt)
	defer cancel()

	if err := cliApp.RunContext(ctx, os.Args); err != nil {
		log.Printf("[error] %s", err)
	}
}

func buildApp(c *cli.Context) (*stefunny.App, error) {
	cfg := stefunny.NewDefaultConfig()
	log.Println("[debug] global config flag", globalConfig)
	log.Println("[debug] local config flag", c.String("config"))
	tfState := c.String("tfstate")
	if tfState == "" {
		tfState = globalTFState
	}
	opt := stefunny.LoadConfigOption{
		TFState: tfState,
	}
	configPath := c.String("config")
	if configPath == "" {
		configPath = globalConfig
	}
	if err := cfg.Load(configPath, opt); err != nil {
		return nil, err
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		return nil, err
	}
	return stefunny.New(c.Context, cfg)
}
