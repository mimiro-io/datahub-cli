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

// deleteCmd represents the delete command
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get dataset with given name",
	Long: `Get a dataset with given name, For example:
mim dataset get --name <name>
or
mim dataset get <name>
`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput = true
		}
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		name, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		if len(args) > 0 {
			name = args[0]
		}
		e, err := getDataset(server, token, name)
		utils.HandleError(err)
		printDataset(e, format)
		pterm.Println()

	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetDatasetsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	GetCmd.Flags().StringP("name", "n", "", "The dataset to get")

}

func getDataset(server string, token string, name string) (*api.Entity, error) {
	res, err := utils.GetRequest(server, token, "/datasets/"+name)
	if err != nil {
		return nil, err
	}

	e := &api.Entity{}
	err = json.Unmarshal(res, e)
	if err != nil {
		return nil, err
	}

	return e, nil
}

func propStripper(entity *api.Entity) map[string]interface{} {
	var singleMap = make(map[string]interface{})
	for k := range entity.Properties {
		singleMap[strings.SplitAfter(k, ":")[1]] = entity.Properties[k]
	}
	return singleMap
}

func printDataset(e *api.Entity, format string) {
	pterm.DefaultSection.Println("Dataset: " + getVal(e.ID))

	stripped := propStripper(e)
	jd, err := json.Marshal(stripped)
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
		out = append(out, []string{"Field", "Value"})
		for k, v := range e.Properties {
			field := getVal(k)
			val := formatField(v)
			if field == "items" {
				val = fmt.Sprintf("%d", int64(v.(float64)))
			}

			out = append(out, []string{
				field,
				val,
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		pterm.Println()
	}
}

func formatField(v interface{}) string {
	switch v.(type) {
	default:
		return fmt.Sprintf("%s", v)
	case int64, int, float64, float32:
		return fmt.Sprintf("%g", v)
	}
}

func getVal(field string) string {
	if strings.Contains(field, ":") {
		res := strings.Split(field, ":")
		return res[1]
	}
	return field
}
