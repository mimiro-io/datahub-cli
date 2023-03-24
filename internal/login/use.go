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
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/display"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
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
		driver := display.ResolveDriver(cmd)

		alias, err := cmd.Flags().GetString("alias")
		driver.RenderError(err, true)

		if alias == "" && len(args) > 0 {
			alias = args[0]
		}

		driver.Msg("")
		driver.RenderSuccess("Setting current login to " + alias)
		_, err = UseLogin(alias)
		driver.RenderError(err, true)
		UpdateConfig(alias)

		pterm.Println()
	},
}

func UseLogin(alias string) (*oauth2.Token, error) {

	// can we login?
	data, err := getLoginAlias(alias)
	if err != nil {
		return nil, err
	}

	// so, we have 3 types of login and some legacy to deal with
	loginType := data.Type
	if data.Type == "" {
		if data.ClientId == "" {
			loginType = "token"
		} else {
			loginType = "client"
		}
	}

	var tkn *oauth2.Token
	switch loginType {
	case "admin":
		tkn, err = web.GetValidToken(data)
		if err != nil {
			tkn, err = web.DoAdminLogin(data)
		}
	case "cert":
		tkn, err = web.GetValidToken(data)
		if err != nil {
			tkn, err = web.GetTokenWithClientCert(data)
		}
	case "client":
		tkn, err = web.GetValidToken(data)
		if err != nil {
			tkn, err = web.DoClientLogin(data)
		}
	case "user":
		tkn, err = web.GetValidToken(data)
		if err != nil {
			lc := NewUserLogin()
			tkn, err = lc.Login(data)
		}
	default:
		tkn = &oauth2.Token{
			AccessToken: data.Token,
		}
	}

	if err != nil {
		return nil, err
	}

	data.OauthToken = tkn
	data.Type = loginType         // this will upgrade existing ones as they are used
	_ = config.Store(alias, data) // don't care about error

	fmt.Println("login success.")
	return tkn, nil
}

func init() {
	UseCmd.Flags().StringP("alias", "a", "", "An alias value for the server")
}
