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

package acl

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get ACL for given client",
	Long: `Get the ACL with given name, For example:
mim acl get <name>
`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		clientId, err := cmd.Flags().GetString("clientId")
		utils.HandleError(err)
		if len(args) > 0 && clientId == "" {
			clientId = args[0]
		}

		if clientId == "" {
			pterm.Warning.Println("You must provide a clientId")
			pterm.Println()
			os.Exit(1)
		}

		pterm.EnableDebugMessages()

		sm := api.NewSecurityManager(server, token)
		clients, err := sm.GetClientAcl(clientId)
		utils.HandleError(err)

		out, err := json.Marshal(clients)
		utils.HandleError(err)
		fmt.Println(string(out))
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetClientsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	GetCmd.Flags().StringP("clientId", "c", "", "The clientId of the client you want to get details about")
}
