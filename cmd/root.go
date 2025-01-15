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
	"fmt"
	"log"
	"os"

	"github.com/akatranlp/go-concurrently/internal/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "go-concurrently",
	Short:   "go-concurrently CLI " + "v0.1.0",
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
		}
		return nil
	},
	SilenceUsage: true,
	RunE: func(ccmd *cobra.Command, args []string) error {
		cfg, err := cmd.ParseConfig()
		if err != nil {
			return err
		}
		log.Printf("%+v", cfg)

		ctx := ccmd.Context()
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for _, command := range cfg.RunBefore.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunBefore.Raw
			}
			sh := cmd.NewCommand(ctx, 0, command.RunCommandConfig)
			if err := sh.Run(nil); err != nil {
				return err
			}
		}

		// Concurrently run all commands

		killOthers := viper.GetBool("killOthers")

		var wg *errgroup.Group
		if killOthers {
			wg, ctx = errgroup.WithContext(ctx)
		} else {
			wg = new(errgroup.Group)
		}
		wg.SetLimit(len(cfg.Commands))

		for i, command := range cfg.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.Raw
			}
			sh := cmd.NewCommand(ctx, i, command)
			wg.Go(func() error {
				defer func() {
					if killOthers {
						cancel()
					}
				}()
				return sh.Run(nil)
			})
		}

		err = wg.Wait()

		for _, command := range cfg.RunAfter.Commands {
			if command.Raw == nil {
				command.Raw = &cfg.RunAfter.Raw
			}
			sh := cmd.NewCommand(context.Background(), 0, command.RunCommandConfig)
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
	fmt.Println(err, "exiting...")
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-concurrently.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("raw", "r", false, "Raw mode (send output of each command directly)")
	viper.BindPFlag("raw", rootCmd.Flags().Lookup("raw"))

	rootCmd.Flags().BoolP("killOthers", "k", false, "Kill all other commands if one fails")
	viper.BindPFlag("killOthers", rootCmd.Flags().Lookup("killOthers"))

	rootCmd.Flags().StringP("beforeCommand", "b", "", "Command to run before all other commands")
	viper.BindPFlag("beforeCommand", rootCmd.Flags().Lookup("beforeCommand"))

	rootCmd.Flags().StringP("afterCommand", "a", "", "Command to run after all other commands")
	viper.BindPFlag("afterCommand", rootCmd.Flags().Lookup("afterCommand"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".go-concurrently")
	}

	// let us see
	// viper.AutomaticEnv() // read in environment variables that match

	_ = viper.ReadInConfig()
}
