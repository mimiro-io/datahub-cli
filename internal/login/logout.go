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
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var LogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Logs out",
	Long: `Logs out from the current profile. To remove the login information you need to delete it instead. For example:
mim logout
`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.EnableDebugMessages()

		UpdateConfig("")

		pterm.Success.Println("Logged out of profile")
		pterm.Println()

	},
}
