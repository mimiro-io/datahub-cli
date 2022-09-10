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
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"os"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// RenameCmd represents the rename command
var RenameCmd = &cobra.Command{
	Use:   "rename",
	Short: "Rename dataset with given id with a new name",
	Long: `Rename a dataset with given id with a new name, For example:
mim dataset rename --name <name> --new-name <newName>
`,
	Run: func(cmd *cobra.Command, args []string) {

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		name, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		newName, err := cmd.Flags().GetString("newName")
		utils.HandleError(err)

		if name == "" || newName == "" {
			pterm.Error.Println("You must provide a dataset name and a new name")
			cmd.Usage()
			os.Exit(1)
		}

		confirm, err := cmd.Flags().GetBool("confirm")
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		dm := api.NewDatasetManager(server, token)

		if confirm {
			pterm.DefaultSection.Printf("Rename dataset with name %s to %s on %s, please type (y)es or (n)o and then press enter:", name, newName, server)
			if utils.AskForConfirmation() {
				err = dm.Rename(name, newName)
				utils.HandleError(err)

				pterm.Success.Println("Renamed dataset")
				pterm.Println()
			} else {
				pterm.Println("Aborted!")
			}
		} else {
			pterm.DefaultSection.Printf("Renaming dataset %s to %s on %s", name, newName, server)
			err = dm.Rename(name, newName)
			utils.HandleError(err)

			pterm.Success.Println("Renamed dataset")
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
	RenameCmd.Flags().StringP("name", "n", "", "The name of the dataset you want to rename")
	RenameCmd.Flags().StringP("newName", "", "", "The new name for the dataset")
	RenameCmd.Flags().BoolP("confirm", "C", true, "Default flag to ask for confirmation before rename")
}
