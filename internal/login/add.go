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
	"errors"

	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/display"
	"github.com/pterm/pterm"
	"github.com/rotisserie/eris"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add or update a login profile",
	Long: `Add (or update) a login profile, for example:
mim login add --server="http://localhost:4242" --token="....secret token here..." --alias="local"
or
mim login add -s https://api.mimiro.io -a prod --clientId="..." --clientSecret="..."
`,
	Run: func(cmd *cobra.Command, args []string) {
		driver := display.ResolveDriver(cmd)
		pterm.EnableDebugMessages()

		// we will attempt to get the server with a token first
		server, _ := cmd.Flags().GetString("server")
		if server == "" {
			driver.RenderError(errors.New("missing server name"), true)
		}

		token, err := cmd.Flags().GetString("token")
		driver.RenderError(err, true)

		// if the alias is set, we use that, if not, we use the server name
		alias, err := cmd.Flags().GetString("alias")
		driver.RenderError(err, true)
		if alias == "" {
			alias = server
		}

		loginType, err := cmd.Flags().GetString("type")
		driver.RenderError(err, true)
		if loginType == "" {
			driver.RenderError(eris.New("you must set a login type. ie. --type user|client|cert|unsecured|admin"), true)
		}

		data := &config.Config{
			Server:       server,
			Token:        "",
			ClientId:     "",
			ClientSecret: "",
			Authorizer:   "",
			Type:         loginType,
		}

		switch loginType {
		case "admin":
			clientId, _ := cmd.Flags().GetString("clientId")
			clientSecret, _ := cmd.Flags().GetString("clientSecret")
			data.ClientId = clientId
			data.ClientSecret = clientSecret
		case "cert":
			clientId, _ := cmd.Flags().GetString("clientId")
			audience, _ := cmd.Flags().GetString("audience")
			data.ClientId = clientId
			data.Audience = audience
		case "client":
			clientId, _ := cmd.Flags().GetString("clientId")
			clientSecret, _ := cmd.Flags().GetString("clientSecret")
			authorizer, _ := cmd.Flags().GetString("authorizer")
			audience, _ := cmd.Flags().GetString("audience")
			if clientSecret == "" {
				driver.RenderError(errors.New("missing client secret"), true)
			}
			if authorizer == "" {
				driver.RenderError(errors.New("missing authorizer url"), true)
			}
			data.ClientId = clientId
			data.ClientSecret = clientSecret
			data.Authorizer = authorizer
			data.Audience = audience
		case "user":
			// this needs only auth server
			authorizer, _ := cmd.Flags().GetString("authorizer")
			data.Authorizer = authorizer
		case "unsecured":
		default:
			data.Token = token // allow empty token
		}

		err = config.Store(alias, data)
		driver.RenderError(err, true)

		driver.Msg("Login added to keyring", "")
	},
	TraverseChildren: true,
}

func init() {
	AddCmd.Flags().StringP("server", "s", "", "The server to add login for")
	AddCmd.Flags().StringP("alias", "a", "", "An alias value for the server")
	AddCmd.Flags().StringP("token", "t", "", "A token to use with the login")
	AddCmd.Flags().StringP("clientId", "", "", "A client id to use in an id/secret pair")
	AddCmd.Flags().StringP("clientSecret", "", "", "A client secret to use in an id/secret pair")
	AddCmd.Flags().StringP("authorizer", "", "", "The authentication server to use with the id/secret")
	AddCmd.Flags().StringP("audience", "", "", "The audience to use for the token")
	AddCmd.Flags().StringP("type", "", "", "One of: admin, client, cert, unsecured or user.")
}
