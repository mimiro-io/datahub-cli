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
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/display"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ListCmd represents the list command
var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "Lists all profiles",
	Long: `Lists all configured login profiles. For example:
mim login list
`,
	Run: func(cmd *cobra.Command, args []string) {
		driver := display.ResolveDriver(cmd)
		pterm.EnableDebugMessages()

		alias := viper.GetString("activelogin")

		var err2 error
		if items, err := config.Dump(); err != nil {
			driver.RenderError(err, true)
		} else {
			out := make([][]string, 0)
			out = append(out, []string{"", "Alias", "Server", "Type", "Token", "ClientId", "ClientSecret", "Authorizer", "Audience", "Subject"})
			for k, v := range items {
				data := &config.Config{}
				err2 = json.Unmarshal(v, data)
				if err2 != nil {
					break
				}
				loginType := data.Type
				if data.Type == "" {
					loginType = "client"
				}

				token := data.Token
				if token != "" {
					token = "*****"
					if data.Type == "" {
						loginType = "token"
					}
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
				if alias == k {
					active = " -> "
				}

				var sub string
				if data.OauthToken != nil {
					at, err := jwt.ParseString(data.OauthToken.AccessToken, jwt.WithVerify(false), jwt.WithValidate(false))
					if err != nil {
						driver.RenderError(err, true)
					}
					sub = at.Subject()
				}

				out = append(out, []string{
					active, k, data.Server, loginType, token, data.ClientId, secret, data.Authorizer, audience, sub,
				})
				out = utils.SortOutputList(out, "Alias")
			}
			driver.Render(out, true)
		}
		driver.RenderError(err2, true)
	},
	TraverseChildren: true,
}

func init() {

}

func GetLoginsCompletion(pattern string) []string {
	var aliases []string
	if items, err := config.Dump(); err != nil {
		utils.HandleError(err)
	} else {
		for k := range items {
			aliases = append(aliases, k)
		}
	}
	return aliases
}
