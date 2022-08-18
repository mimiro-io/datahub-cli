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
	"os"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete job with given job id",
	Long: `Delete a job with given id, For example:
mim jobs delete --id <jobid>
or
mim jobs delete -i <jobid>

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		idOrTitle, err := cmd.Flags().GetString("id")
		utils.HandleError(err)

		if len(args) > 0 {
			// use this as id
			idOrTitle = args[0]
		}

		if idOrTitle == "" {
			pterm.Error.Println("You must provide an job id")
			os.Exit(1)
		}

		confirm, err := cmd.Flags().GetBool("confirm")
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		jm := api.NewJobManager(server, token)

		id := jm.ResolveId(idOrTitle)
		pterm.DefaultSection.Println("Deleting job with id: " + id + " (" + idOrTitle + ") ")

		if confirm {
			pterm.Warning.Printf("Delete job with job id: " + id + " (" + idOrTitle + ") on " + server + ", please type (y)es or (n)o and then press enter:")
			if utils.AskForConfirmation() {
				err = jm.DeleteJob(id)
				utils.HandleError(err)

				pterm.Success.Println("Deleted job")
				pterm.Println()
			} else {
				pterm.Println("Aborted!")
			}
		} else {
			err = jm.DeleteJob(id)
			utils.HandleError(err)

			pterm.Success.Println("Deleted job")
			pterm.Println()
		}
	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	TraverseChildren: true,
}

func init() {

	DeleteCmd.Flags().StringP("id", "i", "", "The name of the job you want to delete")
	DeleteCmd.Flags().BoolP("confirm", "C", true, "Default flag to as for confirmation before delete")
}
