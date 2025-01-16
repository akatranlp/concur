/*
Copyright © 2025 Fabian Petersen <fabian@nf-petersen.de>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/akatranlp/concur/internal/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var commandNames []string
var prefixColors []string
var cfgFile string

const long = `concur is a CLI tool to run multiple commands concurrently;
It can be configured using a configuration file (default: ./.concur.yaml) or by passing commands as arguments.
an example configuration file:

` + "```yaml" + `
commands:
  - command: echo "hello"
    name: hello
  - command: echo "world"
    name: world

runBefore:
  commands:
    - command: echo "before"

runAfter:
  commands:
    - command: echo "after"

` + "```"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "concur",
	Short:   "concur CLI " + "v0.1.0",
	Long:    long,
	Version: "v0.1.0",
	Args:    cobra.ArbitraryArgs,
	PreRunE: func(_ *cobra.Command, args []string) error {
		if len(args) == 0 {
			if cfgFile != "" {
				viper.SetConfigFile(cfgFile)
			} else {
				cwd, err := os.Getwd()
				cobra.CheckErr(err)

				viper.AddConfigPath(cwd)
				viper.SetConfigType("yaml")
				viper.SetConfigName(".concur")
			}

			return viper.ReadInConfig()
		}

		runCfgs := make([]cmd.RunCommandConfig, len(args))
		for i, arg := range args {
			runCfgs[i] = cmd.RunCommandConfig{
				Command: arg,
			}
		}

		if len(commandNames) > 0 {
			if len(commandNames) != len(runCfgs) {
				return errors.New("number of command names must match number of commands")
			}
			for i, name := range commandNames {
				runCfgs[i].Name = name
			}
		}

		if len(prefixColors) > 0 {
			if len(prefixColors) != len(runCfgs) {
				return errors.New("number of prefix colors must match number of commands")
			}
			for i, color := range prefixColors {
				var c cmd.Color
				if err := c.Set(color); err != nil {
					return err
				}
				runCfgs[i].PrefixColor = cmd.Sequence{Color: c}
			}
		}

		viper.Set("runafter", map[string]interface{}{"commands": []interface{}{}})
		viper.Set("commands", runCfgs)
		viper.Set("runbefore", map[string]interface{}{"commands": []interface{}{}})
		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(ccmd *cobra.Command, args []string) error {
		cfg, err := cmd.ParseConfig()
		if err != nil {
			return err
		}

		if cfg.Debug {
			cfg.PrintDebug()
		}

		ctx := ccmd.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if len(cfg.RunBefore.Commands) > 0 {
			fmt.Println("\033[1m[RunBefore]\033[0m")
		}
		for i, command := range cfg.RunBefore.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunBefore.Raw
			}
			prefix, err := cmd.NewPrefix(cfg.Prefix)
			if err != nil {
				panic("unreachable")
			}
			sh := cmd.NewCommand(ctx, i, prefix, command.RunCommandConfig)
			if err := sh.Run(nil); err != nil {
				return err
			}
		}

		fmt.Println("\033[1m[Concurrently]\033[0m")
		// Concurrently run all commands

		killOthers := cfg.KillOthers
		killOthersOnFail := cfg.KillOthersOnFail

		startedCommands := make([]*cmd.Command, len(cfg.Commands))
		for i, command := range cfg.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.Raw
			}
			prefix, err := cmd.NewPrefix(cfg.Prefix)
			if err != nil {
				panic("unreachable")
			}
			sh := cmd.NewCommand(ctx, i, prefix, command)
			if err := sh.Start(); err != nil {
				return err
			}

			startedCommands[i] = sh
		}

		if cfg.PadPrefix {
			cmd.ApplyEvenPadding(startedCommands...)
		}

		var wg sync.WaitGroup
		wg.Add(len(cfg.Commands))
		errCh := make(chan error, len(cfg.Commands))

		for _, sh := range startedCommands {
			go func(cmd *cmd.Command) {
				defer wg.Done()
				err := cmd.Wait(nil)
				if killOthers || (err != nil && killOthersOnFail) {
					cancel()
				}
				errCh <- err
			}(sh)
		}

		wg.Wait()
		close(errCh)
		for errV := range errCh {
			err = errors.Join(err, errV)
		}

		if len(cfg.RunAfter.Commands) > 0 {
			fmt.Println("\033[1m[RunAfter]\033[0m")
		}

		for i, command := range cfg.RunAfter.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunAfter.Raw
			}
			prefix, err := cmd.NewPrefix(cfg.Prefix)
			if err != nil {
				panic("unreachable")
			}
			sh := cmd.NewCommand(context.Background(), i, prefix, command.RunCommandConfig)
			if err := sh.Run(nil); err != nil {
				return err
			}
		}

		if cfg.Debug {
			return err
		}
		return ErrNoPrint{}
	},
}

type ErrNoPrint struct{}

func (ErrNoPrint) Error() string {
	return ""
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteContext(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		if !errors.Is(err, ErrNoPrint{}) {
			fmt.Println("Error:", err)
		}
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.concur.yaml)")

	rootCmd.Flags().StringArrayVarP(&commandNames, "names", "n", nil, "Command names")

	rootCmd.Flags().StringArrayVarP(&prefixColors, "prefix-colors", "c", nil, "Prefix Colors")

	rootCmd.Flags().StringP("prefix", "p", "", "Prefix Type (values: index, name, command, pid, time, TEMPLATE)\n  template Values: {{.Name | .Index | .Command | .Pid | .Time}}")
	viper.BindPFlag("prefix", rootCmd.Flags().Lookup("prefix"))

	rootCmd.Flags().Bool("pad-prefix", false, "Pad prefix to the longest prefix")
	viper.BindPFlag("padPrefix", rootCmd.Flags().Lookup("pad-prefix"))

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("raw", "r", false, "Raw mode (send output of each command directly)")
	viper.BindPFlag("raw", rootCmd.Flags().Lookup("raw"))

	rootCmd.Flags().Bool("debug", false, "Debug mode")
	viper.BindPFlag("debug", rootCmd.Flags().Lookup("debug"))

	rootCmd.Flags().BoolP("kill-others", "k", false, "Kill all other commands if one exists")
	viper.BindPFlag("killOthers", rootCmd.Flags().Lookup("kill-others"))

	rootCmd.Flags().Bool("kill-others-on-fail", false, "Kill all other commands if one fails")
	viper.BindPFlag("killOthersOnFail", rootCmd.Flags().Lookup("kill-others-on-fail"))
}
