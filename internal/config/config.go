package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

var (
	ErrEmptyCommand = errors.New("empty command")
)

type RunCommandConfig struct {
	Command     string   `mapstructure:"command"`
	Name        string   `mapstructure:"name"`
	PrefixColor Sequence `mapstructure:",squash"`
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
}

func (c RunBeforeCommandConfig) Validate() error {
	if err := c.RunCommandConfig.Validate(); err != nil {
		return err
	}

	return nil
}

type RunBeforeConfig struct {
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

type RunAfterCommandConfig struct {
	RunCommandConfig `mapstructure:",squash"`
}

func (c RunAfterCommandConfig) Validate() error {
	if err := c.RunCommandConfig.Validate(); err != nil {
		return err
	}
	return nil
}

type RunAfterConfig struct {
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

type PrefixConfig struct {
	Template        string `mapstructure:"template"`
	PadPrefix       bool   `mapstructure:"padPrefix"`
	PrefixLength    int    `mapstructure:"prefixLength"`
	TimestampFormat string `mapstructure:"timestampFormat"`
	TimeSinceStart  bool   `mapstructure:"timeSinceStart"`
}

type CheckType string

const (
	CheckTypeCommand CheckType = "command"
	CheckTypeHTTP    CheckType = "http"
)

type StatusCheckConfig struct {
	Type     CheckType     `mapstructure:"type"`
	Interval time.Duration `mapstructure:"interval"`
	Command  string        `mapstructure:"command"`
	URL      string        `mapstructure:"url"`
	Template string        `mapstructure:"template"`
}

func (c StatusCheckConfig) Validate() error {
	switch c.Type {
	case CheckTypeCommand:
		if c.Command == "" {
			return ErrEmptyCommand
		}
	case CheckTypeHTTP:
		if c.URL == "" {
			return errors.New("empty URL")
		} else if _, err := url.Parse(c.URL); err != nil {
			return err
		} else if c.Template == "" {
			return errors.New("empty template")
		} else if c.Interval <= 100*time.Millisecond {
			return errors.New("interval too small")
		}
	default:
		return errors.New("invalid check type")
	}
	return nil
}

type StatusConfig struct {
	Enabled       bool                `mapstructure:"enabled"`
	PrintInterval time.Duration       `mapstructure:"printInterval"`
	Sequence      Sequence            `mapstructure:",squash"`
	Text          string              `mapstructure:"text"`
	Checks        []StatusCheckConfig `mapstructure:"checks"`
}

func (c StatusConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if len(c.Checks) == 0 {
		return ErrEmptyCommand
	} else if c.PrintInterval <= 100*time.Millisecond {
		return errors.New("print interval too small")
	}
	for _, check := range c.Checks {
		if err := check.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type Config struct {
	Raw              bool         `mapstructure:"raw"`
	KillOthers       bool         `mapstructure:"killOthers"`
	KillOthersOnFail bool         `mapstructure:"killOthersOnFail"`
	KillSignal       KillSignal   `mapstructure:"killSignal"`
	Debug            bool         `mapstructure:"debug"`
	Prefix           PrefixConfig `mapstructure:"prefix"`
	Commands         []RunCommandConfig
	Status           StatusConfig
	RunBefore        RunBeforeConfig
	RunAfter         RunAfterConfig
}

func (c Config) Validate() error {
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
	if err := c.Status.Validate(); err != nil {
		return err
	}

	return nil
}

func (c Config) PrintDebug() {
	if c.Debug {
		fmt.Println("Viper debug:")
		viper.Debug()
	}

	fmt.Println("Config:")
	fmt.Printf("%+v\n", c)
}

func ParseConfig() (*Config, error) {
	// viper.SetDefault("runBefore.raw", true)
	// viper.SetDefault("runAfter.raw", true)
	viper.SetDefault("status.enabled", false)
	viper.SetDefault("status.text", "HEALTH")
	viper.SetDefault("status.color", "red")
	viper.SetDefault("status.bold", true)
	viper.SetDefault("status.printInterval", 2*time.Second)

	var cfg Config
	if err := viper.Unmarshal(&cfg,
		viper.DecodeHook(
			mapstructure.ComposeDecodeHookFunc(
				mapstructure.StringToTimeDurationHookFunc(),
				mapstructure.TextUnmarshallerHookFunc(),
			),
		),
	); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}
