/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package datahub_mim_cli

import (
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"os"

	"github.com/mimiro-io/datahub-cli/internal/login"

	"github.com/mimiro-io/datahub-cli/internal/command"

	"github.com/spf13/cobra"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "mim",
	Short: "MIMIRO Data Hub CLI",
	Long:  `MIMIRO Data Hub CLI`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().Bool("pretty", false, "Formats the output as pretty where possible")
	RootCmd.PersistentFlags().Bool("json", false, "Formats the output as raw json if possible")
	addCommands()
}

func addCommands() {
	RootCmd.AddCommand(command.JobsCmd)
	RootCmd.AddCommand(command.Login2Cmd)
	RootCmd.AddCommand(login.LogoutCmd)
	RootCmd.AddCommand(command.ContentCmd)
	RootCmd.AddCommand(command.DatasetCmd)
	RootCmd.AddCommand(command.TransformCmd)
	RootCmd.AddCommand(command.QueryCmd)
	RootCmd.AddCommand(command.NamespaceCmd)
	RootCmd.AddCommand(command.CompletionCmd)
	RootCmd.AddCommand(command.TxnsCmd)
	RootCmd.AddCommand(command.AclCmd)
	RootCmd.AddCommand(command.ClientCmd)
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name "/.mim/.datahub-cli" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".datahub-cli")
		viper.SetConfigType("json")
	}

	// this makes it so that viper can read these from the ENV, as it knows about them
	viper.SetDefault("server", "")
	viper.SetDefault("token", "")

	// If a config file is found, read it in.
	_ = viper.ReadInConfig()

	viper.AutomaticEnv() // read in environment variables that match

	// ensure client keys are in place
	err := web.InitialiseClientKeys()
	if err != nil {
		fmt.Println(err)
	}
}
