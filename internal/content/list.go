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

package content

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
	Short:   "List all contents",
	Long: `List all contents. For example:
mim dataset list

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing contents on " + server + "/content")

		contents, err := getContents(server, token)
		utils.HandleError(err)

		err = render(contents, format)
		utils.HandleError(err)
	},
	TraverseChildren: true,
}

func getContents(server string, token string) ([]content, error) {
	contents := make([]content, 0)

	body, err := web.GetRequest(server, token, "/content")
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &contents)
	if err != nil {
		return nil, err
	}

	return contents, nil
}

func render(contents []content, format string) error {

	jd, err := json.Marshal(contents)
	utils.HandleError(err)

	switch format {
	case "json":
		fmt.Println(string(jd))
	case "pretty":
		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	default:
		out := make([][]string, 0)
		out = append(out, []string{"#", "Id"})
		for i, contentObj := range contents {
			out = append(out, []string{
				fmt.Sprintf("%d", i+1),
				contentObj.Id,
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		pterm.Println()
	}
	return nil
}
