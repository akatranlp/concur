package cmd

import (
	"github.com/spf13/viper"
)

type RunCommandConfig struct {
	Command     string      `mapstructure:"command"`
	Name        string      `mapstructure:"name"`
	Raw         *bool       `mapstructure:"raw"`
	PrefixColor prefixColor `mapstructure:"prefixColor"`
	CWD         string      `mapstructure:"cwd"`
	Debug       bool        `mapstructure:"debug"`
}

type InputType string

const (
	InputTypeStdin    InputType = "stdin"
	InputTypePrevious InputType = "previous"
	InputTypeNone     InputType = "none"
)

type OutputType string

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

type RunBeforeConfig struct {
	Raw      bool                     `mapstructure:"raw"`
	Commands []RunBeforeCommandConfig `mapstructure:"commands"`
}

var defaultRunBeforeConfig = RunBeforeConfig{
	Raw: true,
}

type RunAfterCommandConfig struct {
	RunCommandConfig `mapstructure:",squash"`
	Input            InputType  `mapstructure:"input"`
	Output           OutputType `mapstructure:"output"`
}

type RunAfterConfig struct {
	Raw      bool                    `mapstructure:"raw"`
	Commands []RunAfterCommandConfig `mapstructure:"commands"`
}

var defaultRunAfterConfig = RunAfterConfig{
	Raw: true,
}

type Config struct {
	Raw        bool `mapstructure:"raw"`
	KillOthers bool `mapstructure:"killOthers"`
	Commands   []RunCommandConfig
	RunBefore  RunBeforeConfig
	RunAfter   RunAfterConfig
}

func ParseConfig() (*Config, error) {
	var cfg Config
	cfg.RunBefore = defaultRunBeforeConfig
	cfg.RunAfter = defaultRunAfterConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
