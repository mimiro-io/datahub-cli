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

package datasets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/tidwall/pretty"

	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

type datasetName struct {
	Name string `json:"Name"`
}

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all datasets",
	Long: `List all datasets. For example:
mim dataset list

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing datasets on " + server + "/datasets")

		dm := api.NewDatasetManager(server, token)

		sets, err := dm.List()
		utils.HandleError(err)

		renderDataSets(sets, format)
	},
	TraverseChildren: true,
}

func renderDataSets(sets []api.Dataset, format string) {
	switch format {
	case "json":
		out, err := json.Marshal(sets)
		utils.HandleError(err)
		fmt.Println(string(out))
	case "pretty":
		out, err := json.Marshal(sets)
		utils.HandleError(err)
		f := pretty.Pretty(out)
		result := pretty.Color(f, nil)

		fmt.Println(string(result))
	default:
		out := make([][]string, 0)
		out = append(out, []string{"#", "Dir", "Name"})

		for i, set := range sets {
			t := "   "
			if set.Type != nil {
				tt := strings.Join(set.Type, "")
				if strings.Contains(tt, "GET") {
					t = "<"
				} else {
					t = " "
				}
				t = t + "-"
				if strings.Contains(tt, "POST") {
					t = t + ">"
				} else {
					t = t + " "
				}
			}
			out = append(out, []string{
				fmt.Sprintf("%d", i+1),
				t,
				set.Name,
			})
		}

		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	}

	pterm.Println()
}
