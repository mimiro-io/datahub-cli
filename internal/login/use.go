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

package login

import (
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var UseCmd = &cobra.Command{
	Use:   "use",
	Short: "Use a login profile",
	Long: `Uses an already configured login profile. For example:
mim login use local
or
mim login use --alias="dev"
`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.EnableDebugMessages()

		alias, err := cmd.Flags().GetString("alias")
		utils.HandleError(err)

		if alias == "" && len(args) > 0 {
			alias = args[0]
		}

		UseLogin(alias)
		UpdateConfig(alias)

		pterm.Println()
	},
}

func UseLogin(alias string) string {
	pterm.Println()
	pterm.Success.Println("Setting current login to " + alias)

	var token *tokenResponse
	// can we login?
	data, err := getLoginAlias(alias)
	utils.HandleError(err)
	if data.ClientId == "" {
		err = AttemptLogin(data.Server, data.Token)
		utils.HandleError(err)
	} else {
		token, err = exchangeToken(data)
		utils.HandleError(err)

		err = AttemptLogin(data.Server, token.AccessToken)
		utils.HandleError(err)
	}

	if token != nil {
		return token.AccessToken
	}
	return ""
}

func init() {
	UseCmd.Flags().StringP("alias", "a", "", "An alias value for the server")
}
