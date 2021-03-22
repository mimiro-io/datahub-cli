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
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	bolt "go.etcd.io/bbolt"
)

type payload struct {
	Server       string `json:"server"`
	Token        string `json:"token"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Authorizer   string `json:"authorizer"`
	Audience     string `json:"audience"`
}

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

		pterm.EnableDebugMessages()

		// we will attempt to get the server with a token first
		server, _ := cmd.Flags().GetString("server")
		if server == "" {
			utils.HandleError(errors.New("missing server name"))
		}

		token, err := cmd.Flags().GetString("token")
		utils.HandleError(err)

		// if the alias is set, we use that, if not, we use the server name
		alias, err := cmd.Flags().GetString("alias")
		utils.HandleError(err)
		if alias == "" {
			alias = server
		}

		data := &payload{
			Server:       server,
			Token:        "",
			ClientId:     "",
			ClientSecret: "",
			Authorizer:   "",
		}

		clientId, _ := cmd.Flags().GetString("clientId")
		if clientId != "" { // tokens and secrets ar mutually exclusive
			clientSecret, _ := cmd.Flags().GetString("clientSecret")
			authorizer, _ := cmd.Flags().GetString("authorizer")
			audience, _ := cmd.Flags().GetString("audience")
			if clientSecret == "" {
				utils.HandleError(errors.New("missing client secret"))
			}
			if authorizer == "" {
				utils.HandleError(errors.New("missing authorizer url"))
			}
			data.ClientId = clientId
			data.ClientSecret = clientSecret
			data.Authorizer = authorizer
			data.Audience = audience
		} else {
			data.Token = token // allow empty token
		}

		p, err := json.Marshal(data)

		home, err := os.UserHomeDir()
		if _, err := os.Stat(home + "/.mim"); os.IsNotExist(err) { // create dir if not exists
			err = os.Mkdir(home+"/.mim", os.ModePerm)
			utils.HandleError(err)
		}

		db, err := bolt.Open(home+"/.mim/conf.db", 0666, &bolt.Options{Timeout: 1 * time.Second})
		defer db.Close()

		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucketIfNotExists([]byte("logins"))
			if err != nil {
				return err
			}
			return b.Put([]byte(alias), p)
		})
		utils.HandleError(err)

		pterm.Println("Login added to keyring")
		pterm.Println()
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
}
