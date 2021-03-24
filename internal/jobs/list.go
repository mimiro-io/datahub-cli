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
	"strings"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

// listCmd represents the list command
var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List a list of jobs",
	Long: `List a list of jobs. For example:
mim jobs --list
or
mim jobs -l

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput = true
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing server jobs on " + server)

		jobs, err := utils.GetRequest(server, token, "/jobs")
		utils.HandleError(err)

		history, err := utils.GetRequest(server, token, "/jobs/_/history")
		utils.HandleError(err)

		output, err := listJobs(jobs, history)
		utils.HandleError(err)

		printOutput(output, format)

		pterm.Println()
	},
	TraverseChildren: true,
}

func getTransform(job api.Job) string {
	if job.Transform == nil {
		return ""
	}
	if job.Transform["Type"] == nil {
		return ""
	}

	return job.Transform["Type"].(string)
}

func printOutput(output []api.JobOutput, format string) {

	jd, err := json.Marshal(output)
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
		out = append(out, []string{"Id", "Source", "Transform", "Sink", "Paused", "Triggers", "Last Run", "Last Duration", "Error"})

		for _, row := range output {
			lastRun := ""
			lastDuration := ""
			lastError := ""

			if row.History != nil {
				lastRun = row.History.Start.Format(time.RFC3339)
				timed := row.History.End.Sub(row.History.Start)
				lastDuration = fmt.Sprintf("%s", timed)
				lastError = row.History.LastError
				lastError = strings.ReplaceAll(lastError, "\r\n", " ")
				lastError = strings.ReplaceAll(lastError, "\n", " ")
				if len(lastError) > 64 {
					lastError = lastError[:64]
				}
			}

			line := []string{
				row.Job.Id,
				row.Job.Source["Type"].(string),
				getTransform(row.Job),
				row.Job.Sink["Type"].(string),
				fmt.Sprintf("%t", row.Job.Paused)}

			var items []string
			for _, trigger := range row.Job.Triggers {
				jobTypeBullet := ">"
				jobTypeColor := pterm.FgDefault
				if trigger.JobType == "fullsync" {
					jobTypeBullet = ">>"
					jobTypeColor = pterm.FgLightRed
				}
				var item pterm.BulletListItem
				if trigger.TriggerType == "onchange" {
					item = pterm.BulletListItem{
						Level:       0,
						Text:        trigger.MonitoredDataset,
						TextStyle:   pterm.NewStyle(pterm.FgLightBlue),
						Bullet:      jobTypeBullet,
					}
				} else {
					item = pterm.BulletListItem{
						Level:       0,
						Text:        trigger.Schedule,
						TextStyle:   pterm.NewStyle(pterm.FgCyan),
						Bullet:      jobTypeBullet,
						BulletStyle: pterm.NewStyle(jobTypeColor),
					}
				}
				items = append(items, item.Srender())
			}
			triggers := strings.Join(items, ";")
			line = append(line, triggers)

			line = append(line,
				lastRun,
				lastDuration,
				lastError)
			out = append(out, line)
		}
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	}
}

func listJobs(jobs []byte, history []byte) ([]api.JobOutput, error) {
	joblist := &[]api.Job{}
	err := json.Unmarshal(jobs, joblist)
	if err != nil {
		return nil, err
	}

	jobH := &[]api.JobHistory{}
	err = json.Unmarshal(history, jobH)
	if err != nil {
		return nil, err
	}

	hkv := make(map[string]api.JobHistory)
	for _, jh := range *jobH {
		hkv[jh.Id] = jh
	}

	output := make([]api.JobOutput, 0)
	for _, job := range *joblist {
		out := api.JobOutput{
			Job: job,
		}
		if h, ok := hkv[job.Id]; ok {
			out.History = &h
		}
		if h, ok := hkv[job.Id+"_temp"]; ok {
			if out.History == nil || h.Start.After(out.History.Start) {
				out.History = &h
			}
		}
		output = append(output, out)
	}
	return output, nil
}
