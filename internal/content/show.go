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
	"os"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

var ShowCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"show"},
	Short:   "Show a single content",
	Long: `Show a single content. For example:
mim content show --id="my-id"

or

mim content show my-id

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput = true
		}
		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		id, err := cmd.Flags().GetString("id")
		utils.HandleError(err)

		if len(args) > 0 {
			// use this as id
			id = args[0]
		}

		if id == "" {
			pterm.Error.Println("You must provide an id")
			os.Exit(1)
		}

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Describing content on " + server + "/content/" + id)

		content, err := getContent(server, token, id)
		utils.HandleError(err)

		renderContent(content, format)
		utils.HandleError(err)
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return getContentsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	ShowCmd.Flags().StringP("id", "i", "", "The id of the content to look for")
}

func getContentsCompletion(pattern string) []string {
	server, token, err := login.ResolveCredentials()
	utils.HandleError(err)

	contents, err := utils.GetRequest(server, token, "/content")
	utils.HandleError(err)

	contentlist := make([]content, 0)
	err = json.Unmarshal(contents, &contentlist)
	utils.HandleError(err)

	var contentIds []string

	for _, content := range contentlist {
		if strings.HasPrefix(content.Id, pattern) {
			contentIds = append(contentIds, content.Id)
		}
	}
	return contentIds
}

func getContent(server string, token string, id string) (*content, error) {
	body, err := utils.GetRequest(server, token, fmt.Sprintf("/content/%s", id))
	if err != nil {
		return nil, err
	}

	contentObj := &content{}
	err = json.Unmarshal(body, contentObj)
	if err != nil {
		return nil, err
	}

	return contentObj, nil
}

func renderContent(contentObj *content, format string) {
	// we only want the content
	jd, err := json.Marshal(contentObj.Data)
	utils.HandleError(err)

	switch format {
	case "json":
		fmt.Println(string(jd))
	case "pretty":
		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	default:
		pterm.DefaultSection.Println(contentObj.Id)

		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)

		pterm.Println(string(result))
	}
}
