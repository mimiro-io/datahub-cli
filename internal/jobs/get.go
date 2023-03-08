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
	"os"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

// describeCmd represents the describe command
var GetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get job description with given job id",
	Long: `For example:
mim jobs get --id <jobid>
or
mim jobs get -i <jobid>

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
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
		if len(args) > 0 && idOrTitle == "" {
			idOrTitle = args[0]
		}

		if idOrTitle == "" {
			pterm.Warning.Println("You must provide a job title or id")
			pterm.Println()
			os.Exit(1)
		}
		jm := api.NewJobManager(server, token)
		id := jm.ResolveId(idOrTitle)

		pterm.DefaultSection.Printf("Get description of job with id: " + id + " (" + idOrTitle + ") on " + server)

		job, err := jm.GetJob(id)
		utils.HandleError(err)

		renderJob(job, format)

	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {

	GetCmd.Flags().StringP("id", "i", "", "The name of the job you want to get details about")
}

func renderJob(job *api.Job, format string) {

	jd, err := json.Marshal(job)
	utils.HandleError(err)

	switch format {
	case "json":
		fmt.Println(string(jd))
	case "pretty":
		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	default:
		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	}
}
