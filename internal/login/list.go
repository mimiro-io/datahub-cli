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
	"os"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	bolt "go.etcd.io/bbolt"
)

// addCmd represents the add command
var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Lists all profiles",
	Long: `Lists all configured login profiles. For example:
mim login list
`,
	Run: func(cmd *cobra.Command, args []string) {

		pterm.EnableDebugMessages()

		home, err := os.UserHomeDir()
		utils.HandleError(err)

		db, err := bolt.Open(home+"/.mim/conf.db", 0666, &bolt.Options{ReadOnly: true})
		defer db.Close()
		utils.HandleError(err)

		alias := viper.GetString("activelogin")

		out := make([][]string, 0)
		out = append(out, []string{"", "Alias", "Server", "Token", "ClientId", "ClientSecret", "Authorizer", "Audience"})

		err = db.View(func(tx *bolt.Tx) error {
			// Assume bucket exists and has keys

			b := tx.Bucket([]byte("logins"))
			return b.ForEach(func(k, v []byte) error {
				data := &payload{}
				err = json.Unmarshal(v, data)
				if err != nil {
					return err
				}

				token := data.Token
				if token != "" {
					token = "*****"
				}
				secret := data.ClientSecret
				if secret != "" {
					secret = "*****"
				}
				audience := data.Audience
				if audience == "" && data.ClientId != "" {
					audience = data.Server
				}

				active := ""
				if alias == string(k) {
					active = " -> "
				}

				out = append(out, []string{
					active, string(k), data.Server, token, data.ClientId, secret, data.Authorizer, audience,
				})
				return nil
			})

		})
		utils.HandleError(err)
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {

}
