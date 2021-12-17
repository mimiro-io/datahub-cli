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

package jobs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"os"
)

// StatusCmd represents the staus command on a job
var HistoryCmd = &cobra.Command{
	Use:     "history",
	Short:   "history for a job",
	Long:    "history for a job",
	Example: "mim jobs history --id <jobid>",
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		idOrTitle, err := cmd.Flags().GetString("id")
		utils.HandleError(err)
		if idOrTitle == "" && len(args) > 0 {
			idOrTitle = args[0]
		}

		if idOrTitle == "" {
			pterm.Warning.Println("You must provide a job title or id")
			pterm.Println()
			os.Exit(1)
		}

		id := ResolveId(server, token, idOrTitle)

		pterm.DefaultSection.Printf("Get history of job with id: " + id + " (" + idOrTitle + ") on " + server)

		hist := getHistory(id, server, token)
		utils.HandleError(err)

		renderHistory(hist, format)

	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
	TraverseChildren: true,
}

func init() {
	HistoryCmd.Flags().StringP("id", "i", "", "The name of the job you want to get status on")
}

func getHistory(id string, server string, token string) api.JobHistory {
	endpoint := "/jobs/_/history"

	body, err := web.GetRequest(server, token, endpoint)
	utils.HandleError(err)

	histories := make([]api.JobHistory, 0)
	err = json.Unmarshal(body, &histories)
	utils.HandleError(err)

	for _, hist := range histories {
		if hist.Id == id {
			return hist
		}
	}
	return api.JobHistory{}
}

func renderHistory(history api.JobHistory, format string) {
	bf := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(bf)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(history)
	utils.HandleError(err)
	jd := bf.String()

	switch format {
	case "json":
		fmt.Println(jd)
	default:
		p := pretty.Pretty([]byte(jd))
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	}

}
