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
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/pkg/api"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type jobStatus struct {
	JobId    string    `json:"jobId"`
	JobTitle string    `json:"jobTitle"`
	Started  time.Time `json:"started"`
}

// StatusCmd represents the staus command on a job
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Status on jobs",
	Long: `Status on jobs, For example:
mim jobs status --id <jobid>
or
mim jobs status

`,
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

		var id string

		jm := api.NewJobManager(server, token)

		if idOrTitle != "" {
			id = jm.ResolveId(idOrTitle)
			pterm.DefaultSection.Printf("Get status on job with job id: " + id + " (" + idOrTitle + ") on " + server)
		} else {
			pterm.DefaultSection.Printf("Get status on all running jobs on " + server)
			id = ""
		}
		jobs, err := jm.GetJobStatus(id)
		utils.HandleError(err)

		renderBody(jobs, format)

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
	StatusCmd.Flags().StringP("id", "i", "", "The name of the job you want to get status on")
}

func renderBody(jobs []api.JobStatus, format string) {

	jd, err := json.Marshal(jobs)
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
		out = append(out, []string{"Title", "Started"})

		for _, row := range jobs {
			out = append(out, []string{
				row.JobTitle,
				fmt.Sprintf("%s", row.Started),
			})
		}

		out = utils.SortOutputList(out, "Title")

		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		pterm.Println()
	}

}
