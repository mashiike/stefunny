package sffle

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	sfntypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	jsonnet "github.com/google/go-jsonnet"
	gv "github.com/hashicorp/go-version"
	gc "github.com/kayac/go-config"
)

type Config struct {
	RequiredVersion string `yaml:"required_version,omitempty"`
	AWSRegion       string `yaml:"aws_region,omitempty"`

	StateMachine *StateMachineConfig `yaml:"state_machine,omitempty"`

	//private field
	versionConstraints gv.Constraints `yaml:"-,omitempty"`
	dir                string         `yaml:"-,omitempty"`
}

type StateMachineConfig struct {
	Name             string                     `yaml:"name,omitempty"`
	Type             string                     `yaml:"type,omitempty"`
	RoleArn          string                     `yaml:"role_arn,omitempty"`
	Definition       string                     `yaml:"definition,omitempty"`
	Logging          *StateMachineLoggingConfig `yaml:"logging,omitempty"`
	Tracing          *StateMachineTracingConfig `yaml:"tracing,omitempty"`
	stateMachineType sfntypes.StateMachineType  `yaml:"-,omitempty"`
}

type StateMachineLoggingConfig struct {
	Level                string                                `yaml:"level,omitempty"`
	IncludeExecutionData *bool                                 `yaml:"include_execution_data,omitempty"`
	Destination          *StateMachineLoggingDestinationConfig `yaml:"destination,omitempty"`

	logLevel sfntypes.LogLevel `yaml:"-,omitempty"`
}

type StateMachineLoggingDestinationConfig struct {
	LogGroup string `yaml:"log_group,omitempty"`
}

type StateMachineTracingConfig struct {
	Enabled *bool `yaml:"enabled,omitempty"`
}

func (cfg *Config) Load(path string) error {
	if err := gc.LoadWithEnv(cfg, path); err != nil {
		return fmt.Errorf("config load:%w", err)
	}
	cfg.dir = filepath.Dir(path)
	return cfg.Restrict()
}

// Restrict restricts a configuration.
func (cfg *Config) Restrict() error {
	if cfg.RequiredVersion != "" {
		constraints, err := gv.NewConstraint(cfg.RequiredVersion)
		if err != nil {
			return fmt.Errorf("required_version has invalid format: %w", err)
		}
		cfg.versionConstraints = constraints
	}
	if cfg.StateMachine == nil {
		return errors.New("state_machine is required")
	}
	if err := cfg.StateMachine.Restrict(); err != nil {
		return fmt.Errorf("state_machine.%w", err)
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineConfig) Restrict() error {
	if cfg.Name == "" {
		return errors.New("name is required")
	}
	if cfg.RoleArn == "" {
		return errors.New("role_arn is required")
	}
	if cfg.Definition == "" {
		return errors.New("definition is required")
	}

	var err error
	cfg.stateMachineType, err = restrictSFnStateMachineType(cfg.Type)
	if err != nil {
		return fmt.Errorf("type is %w", err)
	}
	if cfg.Logging == nil {
		return errors.New("logging is required")
	}
	if err := cfg.Logging.Restrict(); err != nil {
		return fmt.Errorf("logging.%w", err)
	}
	if cfg.Tracing == nil {
		return errors.New("tracing is required")
	}
	if err := cfg.Tracing.Restrict(); err != nil {
		return fmt.Errorf("tracing.%w", err)
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineLoggingConfig) Restrict() error {

	if cfg.IncludeExecutionData == nil {
		return errors.New("include_execution_data is required")
	}

	var err error
	cfg.logLevel, err = restrictLogLevel(cfg.Level)
	if err != nil {
		return fmt.Errorf("level is %w", err)
	}

	if cfg.logLevel == sfntypes.LogLevelOff {
		return nil
	}

	if cfg.Destination == nil {
		if cfg.logLevel != sfntypes.LogLevelOff {
			return errors.New("destination is required, if log_level is not OFF")
		}
	} else {
		if cfg.logLevel == sfntypes.LogLevelOff {
			log.Println("[warn] set logging.destination but log_level is OFF.")
		}
		if err := cfg.Destination.Restrict(); err != nil {
			return fmt.Errorf("destination.%w", err)
		}
	}

	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineLoggingDestinationConfig) Restrict() error {

	if cfg.LogGroup == "" {
		return errors.New("log_group is required")
	}
	return nil
}

// Restrict restricts a configuration.
func (cfg *StateMachineTracingConfig) Restrict() error {

	if cfg.Enabled == nil {
		return errors.New("enabled is required")
	}
	return nil
}
func restrictSFnStateMachineType(tstr string) (sfntypes.StateMachineType, error) {
	t := sfntypes.StateMachineType(strings.ToUpper(tstr))
	typeValues := t.Values()
	str := "invalid type: please "
	for i, v := range typeValues {
		if t == v {
			return v, nil
		}
		str += string(v)
		if i < len(typeValues)-1 {
			str += ", "
		}
		if len(typeValues) >= 3 && i == len(typeValues)-2 {
			str += "or "
		}
	}
	return "", errors.New(str)
}

func restrictLogLevel(lstr string) (sfntypes.LogLevel, error) {
	l := sfntypes.LogLevel(strings.ToUpper(lstr))
	levelValues := l.Values()
	str := "invalid level: please "
	for i, v := range levelValues {
		if l == v {
			return v, nil
		}
		str += string(v)
		if i < len(levelValues)-1 {
			str += ", "
		}
		if len(levelValues) >= 3 && i == len(levelValues)-2 {
			str += "or "
		}
	}
	return "", errors.New(str)
}

// ValidateVersion validates a version satisfies required_version.
func (cfg *Config) ValidateVersion(version string) error {
	if cfg.versionConstraints == nil {
		log.Println("[warn] required_version is empty. Skip checking required_version.")
		return nil
	}
	versionParts := strings.SplitN(version, "-", 2)
	v, err := gv.NewVersion(versionParts[0])
	if err != nil {
		log.Printf("[warn] Invalid version format \"%s\". Skip checking required_version.", version)
		// invalid version string (e.g. "current") always allowed
		return nil
	}
	if !cfg.versionConstraints.Check(v) {
		return fmt.Errorf("version %s does not satisfy constraints required_version: %s", version, cfg.versionConstraints)
	}
	return nil
}

func NewDefaultConfig() *Config {
	return &Config{
		AWSRegion: os.Getenv("AWS_REGION"),
		StateMachine: &StateMachineConfig{
			Type:             string(sfntypes.StateMachineTypeStandard),
			stateMachineType: sfntypes.StateMachineTypeStandard,
			Logging: &StateMachineLoggingConfig{
				Level:                string(sfntypes.LogLevelOff),
				logLevel:             sfntypes.LogLevelOff,
				IncludeExecutionData: aws.Bool(true),
			},
			Tracing: &StateMachineTracingConfig{
				Enabled: aws.Bool(false),
			},
		},
	}
}

var jsonnetVM = jsonnet.MakeVM()

func (cfg *Config) LoadDefinition() (string, error) {
	path := filepath.Join(cfg.dir, cfg.StateMachine.Definition)
	log.Printf("[debug] try load definition `%s`\n", path)
	bs, err := loadDefinition(path)
	return string(bs), err
}

func loadDefinition(path string) ([]byte, error) {
	switch filepath.Ext(path) {
	case ".jsonnet":
		jsonStr, err := jsonnetVM.EvaluateFile(path)
		if err != nil {
			return nil, err
		}
		return gc.ReadWithEnvBytes([]byte(jsonStr))
	}
	return gc.ReadWithEnv(path)
}
