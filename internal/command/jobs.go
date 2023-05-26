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

package command

import (
	"github.com/mimiro-io/datahub-cli/internal/docs"
	"os"

	"github.com/mimiro-io/datahub-cli/internal/jobs"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// jobsCmd represents the jobs command
var JobsCmd = &cobra.Command{
	Use:     "job",
	Aliases: []string{"jobs"},
	Short:   "Manage datahub jobs from the cli",
	Long: `See available Commands.
Examples:
	min job list
	mim job add -f config.json
	mim job delete <jobid>
	mim job get -id <jobid>
	mim job history -id <jobid>
	mim job status -id <jobid>
	mim job operate <jobid> -o start
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {

	JobsCmd.AddCommand(jobs.ListCmd)
	JobsCmd.AddCommand(jobs.GetCmd)
	JobsCmd.AddCommand(jobs.AddCmd)
	JobsCmd.AddCommand(jobs.DeleteCmd)
	JobsCmd.AddCommand(jobs.CmdOperate())
	JobsCmd.AddCommand(jobs.StatusCmd)
	JobsCmd.AddCommand(jobs.HistoryCmd)

	// TODO: write nice documentation

	JobsCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		pterm.Println()
		result := docs.RenderMarkdown(command, "doc-jobs.md")
		pterm.Println(result)
	})

}
