package cmd

import (
	"errors"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	ErrEmptyCommand = errors.New("empty command")
)

type RunCommandConfig struct {
	Command     string   `mapstructure:"command"`
	Name        string   `mapstructure:"name"`
	Raw         *bool    `mapstructure:"raw"`
	PrefixColor sequence `mapstructure:",squash"`
	CWD         string   `mapstructure:"cwd"`
	Debug       bool     `mapstructure:"debug"`
}

func (c RunCommandConfig) Validate() error {
	if c.Command == "" {
		return ErrEmptyCommand
	}
	return c.PrefixColor.Validate()
}

type InputType string

func (i InputType) Validate() error {
	switch i {
	case InputTypeStdin, InputTypePrevious, InputTypeNone:
		return nil
	}
	return errors.New("invalid input type")
}

const (
	InputTypeStdin    InputType = "stdin"
	InputTypePrevious InputType = "previous"
	InputTypeNone     InputType = "none"
)

type OutputType string

func (i OutputType) Validate() error {
	switch i {
	case OutputTypeStdout, OutputTypePrevious, OutputTypeNone:
		return nil
	}
	return errors.New("invalid input type")
}

const (
	OutputTypeStdout   OutputType = "stdout"
	OutputTypePrevious OutputType = "previous"
	OutputTypeNone     OutputType = "none"
)

type RunBeforeCommandConfig struct {
	RunCommandConfig `mapstructure:",squash"`
	Input            InputType  `mapstructure:"input"`
	Output           OutputType `mapstructure:"output"`
}

func (c RunBeforeCommandConfig) Validate() error {
	if err := c.RunCommandConfig.Validate(); err != nil {
		return err
	}
	if err := c.Input.Validate(); err != nil {
		return err
	}
	if err := c.Output.Validate(); err != nil {
		return err
	}
	return nil
}

type RunBeforeConfig struct {
	Raw      bool                     `mapstructure:"raw"`
	Commands []RunBeforeCommandConfig `mapstructure:"commands"`
}

func (c RunBeforeConfig) Validate() error {
	for _, command := range c.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}
	return nil
}

var defaultRunBeforeConfig = RunBeforeConfig{
	Raw: true,
}

type RunAfterCommandConfig struct {
	RunCommandConfig `mapstructure:",squash"`
	Input            InputType  `mapstructure:"input"`
	Output           OutputType `mapstructure:"output"`
}

func (c RunAfterCommandConfig) Validate() error {
	if err := c.RunCommandConfig.Validate(); err != nil {
		return err
	}
	if err := c.Input.Validate(); err != nil {
		return err
	}
	if err := c.Output.Validate(); err != nil {
		return err
	}
	return nil
}

type RunAfterConfig struct {
	Raw      bool                    `mapstructure:"raw"`
	Commands []RunAfterCommandConfig `mapstructure:"commands"`
}

func (c RunAfterConfig) Validate() error {
	for _, command := range c.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}
	return nil
}

var defaultRunAfterConfig = RunAfterConfig{
	Raw: true,
}

type Config struct {
	Raw              bool       `mapstructure:"raw"`
	KillOthers       bool       `mapstructure:"killOthers"`
	KillOthersOnFail bool       `mapstructure:"killOthersOnFail"`
	Debug            bool       `mapstructure:"debug"`
	PrefixType       prefixType `mapstructure:"prefixType"`
	Commands         []RunCommandConfig
	RunBefore        RunBeforeConfig
	RunAfter         RunAfterConfig
}

func (c Config) Validate() error {
	if err := c.PrefixType.Validate(); err != nil {
		return err
	}
	for _, command := range c.Commands {
		if err := command.Validate(); err != nil {
			return err
		}
	}
	if err := c.RunBefore.Validate(); err != nil {
		return err
	}
	if err := c.RunAfter.Validate(); err != nil {
		return err
	}
	return nil
}

func ParseConfig() (*Config, error) {
	var cfg Config
	cfg.RunBefore = defaultRunBeforeConfig
	cfg.RunAfter = defaultRunAfterConfig
	if err := viper.Unmarshal(&cfg,
		viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()),
	); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
