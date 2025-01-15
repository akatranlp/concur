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
	"log"
	"os"
	"sync"

	"github.com/akatranlp/concur/internal/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "concur",
	Short:   "concur CLI " + "v0.1.0",
	Version: "v0.1.0",
	Args:    cobra.ArbitraryArgs,
	PreRunE: func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			runCfgs := make([]cmd.RunCommandConfig, len(args))
			for i, arg := range args {
				runCfgs[i] = cmd.RunCommandConfig{
					Command: arg,
				}
			}
			viper.Set("commands", runCfgs)
			viper.Set("runAfter", make(map[string]interface{}, 0))
			viper.Set("runBefore", make(map[string]interface{}, 0))
			return nil
		}

		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			cwd, err := os.Getwd()
			cobra.CheckErr(err)

			viper.AddConfigPath(cwd)
			viper.SetConfigType("yaml")
			viper.SetConfigName(".concur")
		}

		// let us see
		// viper.AutomaticEnv() // read in environment variables that match

		return viper.ReadInConfig()
	},
	SilenceUsage: true,
	RunE: func(ccmd *cobra.Command, args []string) error {
		cfg, err := cmd.ParseConfig()
		if err != nil {
			return err
		}

		if cfg.Debug {
			// log.Printf("%+v", cfg)
			log.Printf("%v", cfg)
		}

		ctx := ccmd.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for i, command := range cfg.RunBefore.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunBefore.Raw
			}
			sh := cmd.NewCommand(ctx, i, cfg.PrefixType, command.RunCommandConfig)
			if err := sh.Run(nil); err != nil {
				return err
			}
		}

		// Concurrently run all commands

		killOthers := cfg.KillOthers
		killOthersOnFail := cfg.KillOthersOnFail

		var wg sync.WaitGroup
		wg.Add(len(cfg.Commands))
		errCh := make(chan error, len(cfg.Commands))

		for i, command := range cfg.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.Raw
			}
			sh := cmd.NewCommand(ctx, i, cfg.PrefixType, command)

			go func(cmd *cmd.Command) {
				defer wg.Done()
				err := cmd.Run(nil)
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

		for i, command := range cfg.RunAfter.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunAfter.Raw
			}
			sh := cmd.NewCommand(context.Background(), i, cfg.PrefixType, command.RunCommandConfig)
			if err := sh.Run(nil); err != nil {
				return err
			}
		}

		return err
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func ExecuteContext(ctx context.Context) {
	err := rootCmd.ExecuteContext(ctx)
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.concur.yaml)")

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
