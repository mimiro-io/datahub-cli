// Copyright 2021 MIMIRO AS
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jobs

import (
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/transform"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

// addCmd represents the add command
var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add job based on config",
	Long: `Add a job to server based on config. For example:
mim jobs add -file <configfile.json>
or
mim jobs add -f <configfile.json>

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		file, err := cmd.Flags().GetString("file")
		utils.HandleError(err)

		config, err := utils.ReadInput(file)
		utils.HandleError(err)
		pterm.Success.Println("Read config file")

		jobManager := api.NewJobManager(server, token)

		tfile, err := cmd.Flags().GetString("transform")

		if tfile != "" {
			var code []byte
			importer := transform.NewImporter(tfile)
			if filepath.Ext(tfile) == ".ts"{
				code, err = importer.ImportTs()
			} else {
				code, err = importer.ImportJs()
			}
			utils.HandleError(err)

			job, err := jobManager.AddJob(config)
			utils.HandleError(err)
			pterm.Success.Println("Added job to server")

			job, err = jobManager.AddTransform(job, importer.Encode(code))
			if (err != nil){
				pterm.Error.Println(fmt.Sprintf("Could not add Transform to job. Response from datahub was: %s", err))
				pterm.Println()
				os.Exit(1)
			}
			pterm.Success.Println("Added transform to job")
		} else {
			_, err := jobManager.AddJob(config)
			utils.HandleError(err)
			pterm.Success.Println("Added job to server")
		}

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
	AddCmd.Flags().StringP("file", "f", "", "The job config file to post to create job")
	AddCmd.Flags().StringP("transform", "t", "", "The path to a (local) transform file. The file will be added using a transform import.")
}
