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

package content

import (
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"os"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes a content",
	Long: `Deletes a content. For example:
mim content delete --id="my-id"

or

mim content delete my-id

`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		id, err := cmd.Flags().GetString("id")
		utils.HandleError(err)

		if len(args) > 0 {
			// use this as id
			id = args[0]
		}

		if id == "" {
			pterm.Error.Println("You must provide an id")
			os.Exit(1)
		}

		confirm, err := cmd.Flags().GetBool("confirm")
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Deleting content " + server + "/content/" + id)

		if confirm {
			pterm.DefaultSection.Printf("Delete content with content id " + id + " on " + server + ", please type (y)es or (n)o and then press enter:")
			if utils.AskForConfirmation() {
				err = web.DeleteRequest(server, token, fmt.Sprintf("/content/%s", id))
				utils.HandleError(err)

				pterm.Success.Println("Deleted content")
				pterm.Println()
			} else {
				pterm.Println("Aborted!")
			}
		} else {
			err = web.DeleteRequest(server, token, fmt.Sprintf("/content/%s", id))
			utils.HandleError(err)

			pterm.Success.Println("Deleted content")
			pterm.Println()
		}

	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getContentsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	DeleteCmd.Flags().StringP("id", "i", "", "The id of the content to delete.")
	DeleteCmd.Flags().BoolP("confirm", "C", true, "Default flag to ask for confirmation before delete")
}
