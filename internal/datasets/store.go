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
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/mimiro-io/datahub-cli/pkg/api"
	"io/ioutil"

	"github.com/mimiro-io/datahub-cli/internal/login"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var StoreCmd = &cobra.Command{
	Use:   "store",
	Short: "Store entities in dataset with given name",
	Long: `Store entities in a dataset with given name, For example:
mim dataset store --name=<name> --file=entities.json
or
mim dataset create <name> entities.json
`,
	Run: func(cmd *cobra.Command, args []string) {
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		name, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		filename, err := cmd.Flags().GetString("filename")
		utils.HandleError(err)

		if len(args) == 1 {
			name = args[0]
		}
		if len(args) == 2 {
			name = args[0]
			filename = args[1]
		}

		err = storeEntities(server, token, name, filename)
		utils.HandleError(err)
		pterm.Success.Println("Entities Loaded")
		pterm.Println()

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
	StoreCmd.Flags().StringP("name", "n", "", "The name of the dataset to create")
	StoreCmd.Flags().StringP("filename", "f", "", "The name of the file to load entities from")
}

func storeEntities(server string, token string, name string, filename string) error {

	// load data from file as bytes
	entitiesJSON, err := ioutil.ReadFile(filename)

	_, err = web.PostRequest(server, token, "/datasets/"+name+"/entities", entitiesJSON)
	if err != nil {
		return err
	}

	return nil
}
