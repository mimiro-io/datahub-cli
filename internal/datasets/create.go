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
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var CreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create dataset with given name",
	Long: `Create a dataset with given name, For example:
mim dataset create --name <name>
or
mim dataset create <name>
`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		name, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		if len(args) > 0 {
			name = args[0]
		}

		err = updateDataset(server, token, name)
		utils.HandleError(err)
		pterm.Success.Println("Dataset has been created")
		pterm.Println()

	},
	TraverseChildren: true,
}

func init() {
	CreateCmd.Flags().StringP("name", "n", "", "The dataset to create")

}

func updateDataset(server string, token string, name string) error {

	_, err := utils.PostRequest(server, token, "/datasets/"+name, nil)
	if err != nil {
		return err
	}

	return nil

}
