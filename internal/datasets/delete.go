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

package datasets

import (
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
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
	Short: "Delete dataset with given id",
	Long: `Delete a dataset with given  id, For example:
mim datasets delete --id <id>
or
mim jobs delete -i <id>
`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		name, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		if len(args) > 0 {
			// use this as name
			name = args[0]
		}

		if name == "" {
			pterm.Error.Println("You must provide a dataset name")
			os.Exit(1)
		}

		confirm, err := cmd.Flags().GetBool("confirm")
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Deleting dataset " + server + "/datasets/" + name)

		if confirm {
			pterm.DefaultSection.Printf("Delete dataset with name " + name + " on " + server + ", please type (y)es or (n)o and then press enter:")
			if utils.AskForConfirmation() {
				err = web.DeleteRequest(server, token, fmt.Sprintf("/datasets/%s", name))
				utils.HandleError(err)

				pterm.Success.Println("Deleted dataset")
				pterm.Println()
			} else {
				pterm.Println("Aborted!")
			}
		} else {
			err = web.DeleteRequest(server, token, fmt.Sprintf("/datasets/%s", name))
			utils.HandleError(err)

			pterm.Success.Println("Deleted dataset")
			pterm.Println()
		}
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetDatasetsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {

	DeleteCmd.Flags().StringP("name", "n", "", "The name of the dataset you want to get delete")
	DeleteCmd.Flags().BoolP("confirm", "C", true, "Default flag to as for confirmation before delete")
}
