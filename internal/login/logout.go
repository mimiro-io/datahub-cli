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
	"github.com/mimiro-io/datahub-cli/internal/config"
	"github.com/mimiro-io/datahub-cli/internal/display"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logs out",
	Long: `Logs out from the current profile. To remove the login information you need to delete it instead. For example:
mim logout
`,
	Run: func(cmd *cobra.Command, args []string) {
		driver := display.ResolveDriver(cmd)

		err := removeToken()
		driver.RenderError(err, true)

		UpdateConfig("")

		driver.RenderSuccess("Logged out of profile")
	},
}

func removeToken() error {
	alias := viper.GetString("activelogin")

	if alias != "" {
		data := &config.Config{}
		if err := config.Load(alias, data); err != nil {
			return err
		}

		data.OauthToken = nil
		if err := config.Store(alias, data); err != nil {
			return err
		}
	}

	return nil
}
