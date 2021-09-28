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
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing server jobs on " + server)

		jobs, err := utils.GetRequest(server, token, "/jobs")
		utils.HandleError(err)

		history, err := utils.GetRequest(server, token, "/jobs/_/history")
		utils.HandleError(err)

		filter, err := cmd.Flags().GetString("filter")
		if filter == "" && len(args) > 0 {
			filter = args[0]
		}
		output, err := listJobs(jobs, history)
		utils.HandleError(err)

		if filter != ""{
			output = filterJobs(output, filter)
		}

		verbose, err := cmd.Flags().GetBool("verbose")
		if verbose {
			printOutput(output, "verbose")
		}else{
			printOutput(output, format)
		}

		pterm.Println()
	},
	TraverseChildren: true,
}

func filterJobs(jobOutputs []api.JobOutput, filters string) []api.JobOutput {

	var tags []string
	var titles []string
	if strings.Contains(filters, "tags=") || strings.Contains(filters, "tag="){
		tags = strings.Split(strings.Trim(filters, "tags="),",")
	}

	if strings.Contains(filters, "title="){
		titles = strings.Split(strings.Trim(filters, "title="),",")
	}
	filterList := strings.Split(filters, ",")

	output := make([]api.JobOutput, 0)

	for _, jobOutput := range jobOutputs {
		id := jobOutput.Job.Id
		title := jobOutput.Job.Title
		jobTags := jobOutput.Job.Tags
		if titles != nil{ //searches for titles and ids containing the filter
			for _, filter := range titles{
				if strings.Contains(id, filter) || strings.Contains(title, filter){
					if !objectContains(output, jobOutput.Job.Id) {
						output = append(output, jobOutput)
					}
				}
			}
		} else if tags != nil { //searches for tags containing the filter
			for _, filter := range tags{
				if listContains(jobTags,filter){
					if !objectContains(output, jobOutput.Job.Id) {
						output = append(output, jobOutput)
					}
				}
			}
		}else { //searches for titles, ids and tags containing the filter
			for _, filter := range filterList{
				if strings.Contains(id, filter) || strings.Contains(title, filter){
					if !objectContains(output, jobOutput.Job.Id) {
						output = append(output, jobOutput)
					}
				}
				if listContains(jobTags,filter){
					if !objectContains(output, jobOutput.Job.Id) {
						output = append(output, jobOutput)
					}
				}
			}
		}
	}
	return output
}

func objectContains(object []api.JobOutput, id string ) bool{
	exists := false
	for _, o := range object{
		if o.Job.Id == id {
			exists = true
		}
	}
	return exists
}


func listContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
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
	case "verbose":
		out := buildOutput(output, format)
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	default:
		out := buildOutput(output, format)
		pterm.DefaultTable.WithHasHeader().WithData(out).Render()
	}
}

func buildOutput(output []api.JobOutput, format string) [][]string{
	out := make([][]string, 0)
	if format == "verbose"{
		out = append(out, []string{"Id", "Title", "Paused",  "Tags", "Source", "Transform", "Sink", "Triggers", "Last Run", "Last Duration", "Error"})
	} else {
		out = append(out, []string{"Title", "Paused",  "Tags", "Source", "Transform", "Sink", "Last Run", "Last Duration", "Error"})
	}


	for _, row := range output {
		lastRun := ""
		lastDuration := ""
		lastError := ""
		title := ""
		tags := ""

		if row.History != nil {
			lastRun = row.History.Start.Format(time.RFC3339)
			timed := row.History.End.Sub(row.History.Start)
			lastDuration = fmt.Sprintf("%s", timed)
			lastError = row.History.LastError
			lastError = strings.ReplaceAll(lastError, "\r\n", " ")
			lastError = strings.ReplaceAll(lastError, "\n", " ")
			if len(lastError) > 30 {
				lastError = lastError[:30]+"..."
			}
		}

		title = row.Job.Title
		if title == ""{
			title = row.Job.Id
		}

		pausedColor := pterm.FgLightRed
		if row.Job.Paused == false{
			pausedColor = pterm.FgDefault
		}
		var pausedItem = pterm.BulletListItem{
			Level:       0,
			Text:        fmt.Sprintf("%t", row.Job.Paused),
			TextStyle:   pterm.NewStyle(pausedColor),
			Bullet:      "",
		}
		pausedString, err := pterm.BulletListPrinter{}.WithItems([]pterm.BulletListItem{pausedItem}).Srender()
		utils.HandleError(err)

		if len(row.Job.Tags) > 0{
			tags = strings.Join(row.Job.Tags, ",")
		}
		source := row.Job.Source["Type"].(string)
		if source == "DatasetSource"{
			source = "Dataset"
		}
		if source == "HttpDatasetSource"{
			source = "Http"
		}
		sink := row.Job.Sink["Type"].(string)
		if sink == "DatasetSink"{
			sink = "Dataset"
		}
		if sink == "HttpDatasetSink"{
			sink = "Http"
		}
		transform := getTransform(row.Job)
		if transform == "JavascriptTransform"{
			transform = "Javascript"
		}
		//line output for each row
		var line  []string
		if format == "verbose"{
			line = append(line, row.Job.Id)
		}
		line = append(line, title, strings.TrimSpace(pausedString), tags, source, transform, sink)

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
			itemString, err := pterm.BulletListPrinter{}.WithItems([]pterm.BulletListItem{item}).Srender()
			utils.HandleError(err)
			items = append(items, strings.TrimSpace(itemString))
		}
		if format == "verbose"{
			triggers := strings.Join(items, ";")
			line = append(line, triggers)
		}

		line = append(line,
			lastRun,
			lastDuration,
			lastError)
		out = append(out, line)
	}
	return out
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


func ResolveId(server string, token string, title string) string {
	id := title
	allJobs, err := utils.GetRequest(server, token, "/jobs")
	utils.HandleError(err)

	joblist := &[]api.Job{}
	err = json.Unmarshal(allJobs, joblist)
	if err != nil {
		return title
	}

	for _, job := range *joblist {
		out := api.JobOutput{
			Job: job,
		}
		if out.Job.Title == title{
			id = out.Job.Id
		}
	}
	return id
}


func init() {
	ListCmd.PersistentFlags().Bool("verbose", false, "Verbose output of jobs list")
	ListCmd.PersistentFlags().StringP("filter", "","", "Filter in all jobs with a comma separated string i.e  'tags=foo,bar' or 'title=foo,bar'. '--filter foo,bar' gives you a result set across titles and tags" )
}
