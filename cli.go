package stefunny

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
)

const dryRunStr = "DRY RUN"

type CLI struct {
	LogLevel  string   `name:"log-level" help:"Set log level (debug, info, notice, warn, error)" default:"info" env:"STEFUNNY_LOG_LEVEL" json:"log_level,omitempty"`
	Config    string   `name:"config" short:"c" help:"Path to config file" default:"config.yaml" env:"STEFUNNY_CONFIG" json:"config,omitempty"`
	TFState   string   `name:"tfstate" help:"URL to terraform.tfstate referenced in config" env:"STEFUNNY_TFSTATE" json:"tfstate,omitempty"`
	ExtStr    []string `name:"ext-str" help:"external string values for Jsonnet" default:"" json:"ext_str,omitempty"`
	ExtCode   []string `name:"ext-code" help:"external code values for Jsonnet" default:"" json:"ext_code,omitempty"`
	AWSRegion string   `name:"region" help:"AWS region" default:"" env:"AWS_REGION" json:"region,omitempty"`

	Version  struct{}              `cmd:"" help:"Show version" json:"version,omitempty"`
	Init     InitOption            `cmd:"" help:"Initialize stefunny configuration" json:"init,omitempty"`
	Delete   DeleteOption          `cmd:"" help:"Delete state machine and schedule rules" json:"delete,omitempty"`
	Deploy   DeployCommandOption   `cmd:"" help:"Deploy state machine and schedule rules" json:"deploy,omitempty"`
	Schedule ScheduleCommandOption `cmd:"" help:"Enable or disable schedule rules" json:"schedule,omitempty"`
	Render   RenderOption          `cmd:"" help:"Render state machine definition" json:"render,omitempty"`
	Execute  ExecuteOption         `cmd:"" help:"Execute state machine" json:"execute,omitempty"`

	kctx           *kong.Context
	exitFunc       func(int)
	stderr, stdout io.Writer
	namedMappers   map[string]kong.Mapper
	setLogLevel    func(string) error
}

func NewCLI() *CLI {
	return &CLI{
		exitFunc: os.Exit,
		stderr:   os.Stderr,
		stdout:   os.Stdout,
		setLogLevel: func(string) error {
			return nil
		},
		namedMappers: map[string]kong.Mapper{},
		Render: RenderOption{
			Writer: os.Stdout,
		},
		Execute: ExecuteOption{
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
	}
}

// Writers sets the writers for stdout and stderr. for testing
func (cli *CLI) Writers(stdout, stderr io.Writer) {
	cli.stdout = stdout
	cli.stderr = stderr
	cli.Render.Writer = stdout
}

// Exit sets the exit function. for testing
func (cli *CLI) Exit(exitFunc func(int)) {
	cli.exitFunc = exitFunc
}

// NoExpandPath disables path expansion. for testing
func (cli *CLI) NoExpandPath() {
	cli.namedMappers["path"] = kong.MapperFunc(
		func(ctx *kong.DecodeContext, target reflect.Value) error {
			var path string
			err := ctx.Scan.PopValueInto("file", &path)
			if err != nil {
				return err
			}
			target.SetString(path)
			return nil
		},
	)
}

func (cli *CLI) SetLogLevelFunc(f func(string) error) {
	cli.setLogLevel = f
}

// Parse parses the command line arguments and returns the command name
func (cli *CLI) Parse(args []string) (string, error) {
	kongOpts := []kong.Option{
		kong.Vars{"version": Version},
		kong.Name("stefunny"),
		kong.Description("stefunny is a deployment tool for AWS StepFunctions state machine"),
		kong.UsageOnError(),
		kong.Exit(cli.exitFunc),
		kong.Writers(cli.stdout, cli.stderr),
	}
	for k, v := range cli.namedMappers {
		kongOpts = append(kongOpts, kong.NamedMapper(k, v))
	}
	parser, err := kong.New(
		cli,
		kongOpts...,
	)
	if err != nil {
		return "", err
	}
	kctx, err := parser.Parse(args)
	if err != nil {
		parser.FatalIfErrorf(err)
		return "", err
	}
	if err := cli.setLogLevel(cli.LogLevel); err != nil {
		return "", fmt.Errorf("failed to set log level: %w", err)
	}
	cli.kctx = kctx
	cmdStr := kctx.Command()
	if cmdStr == "" {
		return "", fmt.Errorf("no command")
	}
	cmd := strings.Fields(cmdStr)[0]
	if cmd == "version" {
		fmt.Fprintf(cli.stdout, "stefunny %s\n", Version)
		kctx.Exit(0)
	}
	return cmd, nil
}

// NewApp creates a new App instance from the CLI configuration
func (cli *CLI) NewApp(ctx context.Context) (*App, error) {
	cfg := NewDefaultConfig()
	log.Println("[debug] config flag", cli.Config)
	extStr := make(map[string]string)
	for _, s := range cli.ExtStr {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid external string value: %s", s)
		}
		extStr[kv[0]] = kv[1]
	}
	extCode := make(map[string]string)
	for _, s := range cli.ExtCode {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid external code value: %s", s)
		}
		extCode[kv[0]] = kv[1]
	}
	opt := LoadConfigOption{
		TFState: cli.TFState,
		ExtStr:  extStr,
		ExtCode: extCode,
	}
	if err := cfg.Load(cli.Config, opt); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.ValidateVersion(Version); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	app, err := New(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create app: %w", err)
	}
	return app, nil
}

// Run() runs the command
func (cli *CLI) Run(ctx context.Context, args []string) error {
	cmd, err := cli.Parse(args)
	if err != nil {
		return err
	}
	app, err := cli.NewApp(ctx)
	if err != nil {
		return err
	}
	switch cmd {
	case "init":
		cli.Init.ConfigPath = cli.Config
		cli.Init.AWSRegion = cli.AWSRegion
		return app.Init(ctx, cli.Init)
	case "deploy":
		return app.Deploy(ctx, cli.Deploy.DeployOption())
	case "schedule":
		return app.Deploy(ctx, cli.Schedule.DeployOption())
	case "delete":
		return app.Delete(ctx, cli.Delete)
	case "render":
		cli.Render.Writer = cli.stdout
		return app.Render(ctx, cli.Render)
	case "execute":
		return app.Execute(ctx, cli.Execute)
	default:
		return fmt.Errorf("unknown command: %s", cmd)
	}
}
