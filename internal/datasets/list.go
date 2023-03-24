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
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/mimiro-io/datahub-cli/pkg/api"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"strings"

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

		resp, err := web.GetRequest(server, token, "/datasets/core.Dataset/entities")

		var coreDataset []api.Entity

		if err == nil {
			json.Unmarshal(resp, &coreDataset)
		}

		if coreDataset != nil {
			merged := mergeDatasetDetails(sets, coreDataset)
			renderDataSets(merged, format)
		} else {
			renderDataSets(sets, format)
		}
	},
	TraverseChildren: true,
}

func mergeDatasetDetails(datasets []api.Dataset, coreDataset []api.Entity) []api.Dataset {
	for i, dataset := range datasets {
		for _, entity := range coreDataset {
			if dataset.Name == entity.ID[4:] {
				datasets[i].Items = int(entity.Properties["ns0:items"].(float64))
			}
		}
	}
	return datasets
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
		out = append(out, []string{"#", "Dir", "Items", "Name"})

		p := message.NewPrinter(language.English)

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
				p.Sprintf("%13d", set.Items),
				set.Name,
			})
		}

		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	}

	pterm.Println()
}
