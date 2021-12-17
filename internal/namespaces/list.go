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

package namespaces

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List a list of jobs",
	Long: `List a list of jobs. For example:
mim jobs --list
or
mim jobs -l
.`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()

		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing server namespaces on " + server)

		namespaces, err := web.GetRequest(server, token, "/namespaces")
		utils.HandleError(err)

		output(namespaces, format)

		pterm.Println()
	},
	TraverseChildren: true,
}

func output(namespaces []byte, format string) {
	switch format {
	case "pretty":
		f := pretty.Pretty(namespaces)
		result := pretty.Color(f, nil)
		fmt.Println(string(result))
	case "json":
		fmt.Println(string(namespaces))
	default:
		out := make([][]string, 0)
		out = append(out, []string{"Prefix", "Expansion"})
		result := make(map[string]string)
		err := json.Unmarshal(namespaces, &result)
		utils.HandleError(err)

		for k, v := range result {
			out = append(out, []string{
				k, v,
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	}

}
