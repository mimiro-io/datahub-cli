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
	"time"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

type jobStatus struct {
	JobId   string    `json:"jobId"`
	Started time.Time `json:"started"`
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

		id, err := cmd.Flags().GetString("id")
		utils.HandleError(err)
		if id == "" && len(args) > 0 {
			id = args[0]
		}

		if id != "" {
			pterm.DefaultSection.Printf("Get status on job with job id " + id + " on " + server)
		} else {
			pterm.DefaultSection.Printf("Get status on all running jobs on " + server)
		}
		jobs, err := getStatus(id, server, token)
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

func getStatus(id string, server string, token string) ([]jobStatus, error) {
	endpoint := "/jobs/_/status"
	if id != "" {
		endpoint = fmt.Sprintf("/job/%s/status", id)
	}

	body, err := utils.GetRequest(server, token, endpoint)
	if err != nil {
		return nil, err
	}

	jobs := make([]jobStatus, 0)
	err = json.Unmarshal(body, &jobs)
	if err != nil {
		return nil, err
	}

	return jobs, nil
}

func renderBody(jobs []jobStatus, format string) {

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
		out = append(out, []string{"Id", "Started"})

		for _, row := range jobs {
			out = append(out, []string{
				row.JobId,
				fmt.Sprintf("%s", row.Started),
			})
		}
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
		pterm.Println()
	}

}
