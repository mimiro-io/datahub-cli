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
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
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

		filterMode, err := cmd.Flags().GetString("filterMode")
		utils.HandleError(err)

		output, err := listJobs(jobs, history)
		utils.HandleError(err)

		if filter != "" {
			output, err = filterJobs(output, filter, filterMode)
			utils.HandleError(err)
		}

		verbose, err := cmd.Flags().GetBool("verbose")
		if verbose {
			printOutput(output, "verbose")
		} else {
			printOutput(output, format)
		}

		pterm.Println()
	},
	TraverseChildren: true,
}

type jobFilter struct {
	jobProperty string
	operator    string
	pattern     []string
}

func filterJobs(jobOutputs []api.JobOutput, filters string, filterMode string) ([]api.JobOutput, error) {

	pattern, _ := regexp.Compile("(\\w+)([=><])((?:[A-Za-z0-9-_.:@+ ]+[,]?)+);?")
	matches := pattern.FindAllStringSubmatch(filters, -1)
	var sortedFilters []jobFilter

	output := make([]api.JobOutput, 0)

	if matches == nil {
		return output, errors.New("unable to parse filter query")
	} else {
		for _, match := range matches {
			sortedFilters = append(sortedFilters, jobFilter{jobProperty: strings.ToLower(match[1]), operator: match[2], pattern: strings.Split(match[3], ",")})
		}
	}

	if sortedFilters != nil {
		if filterMode == "inclusive" {
			for _, filter := range sortedFilters {
				output = append(output, processFilter(jobOutputs, filter)...)
			}
		} else {
			output = jobOutputs
			for _, filter := range sortedFilters {
				output = processFilter(output, filter)
			}
		}
	}
	return output, nil
}

func processFilter(jobList []api.JobOutput, filter jobFilter) []api.JobOutput {
	var output []api.JobOutput
	for _, jobOutput := range jobList {
		switch filter.jobProperty {
		case "id", "title":
			if matchProperty(jobOutput.Job.Id, filter.pattern) {
				output = appendJobOutput(output, jobOutput)
			}
			if matchProperty(jobOutput.Job.Title, filter.pattern) {
				output = appendJobOutput(output, jobOutput)
			}
		case "paused":
			parsedBool, _ := strconv.ParseBool(filter.pattern[0])
			if jobOutput.Job.Paused == parsedBool {
				output = appendJobOutput(output, jobOutput)
			}
		case "tag", "tags":
			for _, tag := range jobOutput.Job.Tags {
				if matchProperty(tag, filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
			}
		case "source":
			if matchProperty(jobOutput.Job.Source["Type"].(string), filter.pattern) {
				output = appendJobOutput(output, jobOutput)
			}
		case "sink":
			if matchProperty(jobOutput.Job.Sink["Type"].(string), filter.pattern) {
				output = appendJobOutput(output, jobOutput)
			}
		case "transform":
			transform := jobOutput.Job.Transform
			if transform != nil {
				if matchProperty(jobOutput.Job.Transform["Type"].(string), filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
			}
		case "error":
			if jobOutput.History != nil {
				if matchProperty(jobOutput.History.LastError, filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
			}
		case "duration":
			if jobOutput.History != nil {
				lastDuration := jobOutput.History.End.Sub(jobOutput.History.Start)
				inputDuration, err := time.ParseDuration(filter.pattern[0])
				if err != nil {
					utils.HandleError(errors.New("unable to parse duration filter"))
				}
				switch filter.operator {
				case "<":
					if lastDuration < inputDuration {
						output = appendJobOutput(output, jobOutput)
					}
				case ">":
					if lastDuration > inputDuration {
						output = appendJobOutput(output, jobOutput)
					}
				}
			}
		case "lastrun":
			if jobOutput.History != nil {
				lastRun := jobOutput.History.Start
				inputTimestamp, err := time.Parse("2006-01-02T15:04:05-07:00", filter.pattern[0])
				if err != nil {
					utils.HandleError(errors.New("unable to parse duration filter"))
				}
				switch filter.operator {
				case "<":
					if lastRun.Before(inputTimestamp) {
						output = appendJobOutput(output, jobOutput)
					}
				case ">":
					if lastRun.After(inputTimestamp) {
						output = appendJobOutput(output, jobOutput)
					}
				}
			}
		case "triggers", "trigger":
			for _, trigger := range jobOutput.Job.Triggers {
				if matchProperty(trigger.Schedule, filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
				if matchProperty(trigger.MonitoredDataset, filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
				if matchProperty(trigger.JobType, filter.pattern) {
					output = appendJobOutput(output, jobOutput)
				}
			}
		}
	}
	return output
}

func matchProperty(property string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(strings.ToLower(property), strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func appendJobOutput(outputArray []api.JobOutput, output api.JobOutput) []api.JobOutput {
	for _, jobOutput := range outputArray {
		if jobOutput.Job.Id == output.Job.Id {
			return outputArray
		}
	}
	return append(outputArray, output)
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

func buildOutput(output []api.JobOutput, format string) [][]string {
	out := make([][]string, 0)
	if format == "verbose" {
		out = append(out, []string{"Id", "Title", "Paused", "Tags", "Source", "Transform", "Sink", "Triggers", "Last Run", "Last Duration", "Error"})
	} else {
		out = append(out, []string{"Title", "Paused", "Tags", "Source", "Transform", "Sink", "Last Run", "Last Duration", "Error"})
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
				lastError = lastError[:30] + "..."
			}
		}

		title = row.Job.Title
		if title == "" {
			title = row.Job.Id
		}

		pausedColor := pterm.FgLightRed
		if row.Job.Paused == false {
			pausedColor = pterm.FgDefault
		}
		var pausedItem = pterm.BulletListItem{
			Level:     0,
			Text:      fmt.Sprintf("%t", row.Job.Paused),
			TextStyle: pterm.NewStyle(pausedColor),
			Bullet:    "",
		}
		pausedString, err := pterm.BulletListPrinter{}.WithItems([]pterm.BulletListItem{pausedItem}).Srender()
		utils.HandleError(err)

		if len(row.Job.Tags) > 0 {
			tags = strings.Join(row.Job.Tags, ",")
		}
		source := row.Job.Source["Type"].(string)
		if source == "DatasetSource" {
			source = "Dataset"
		}
		if source == "HttpDatasetSource" {
			source = "Http"
		}
		sink := row.Job.Sink["Type"].(string)
		if sink == "DatasetSink" {
			sink = "Dataset"
		}
		if sink == "HttpDatasetSink" {
			sink = "Http"
		}
		transform := getTransform(row.Job)
		if transform == "JavascriptTransform" {
			transform = "Javascript"
		}
		//line output for each row
		var line []string
		if format == "verbose" {
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
					Level:     0,
					Text:      trigger.MonitoredDataset,
					TextStyle: pterm.NewStyle(pterm.FgLightBlue),
					Bullet:    jobTypeBullet,
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
		if format == "verbose" {
			triggers := strings.Join(items, ";")
			line = append(line, triggers)
		}

		line = append(line,
			lastRun,
			lastDuration,
			lastError)
		out = append(out, line)
	}

	//sorting the list since it is sorted on ids, but we want it to sort on titles
	sort.Slice(out[:], func(i, j int) bool {
		for x := range out[i] {
			if out[i][x] == out[j][x] {
				continue
			}
			return out[i][x] < out[j][x]
		}
		return false
	})

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
		if out.Job.Title == title {
			id = out.Job.Id
		}
	}
	return id
}

func init() {
	ListCmd.PersistentFlags().Bool("verbose", false, "Verbose output of jobs list")
	ListCmd.PersistentFlags().StringP("filter", "", "", "Filter job list with a filter query i.e  'tags=foo,bar' or 'title=foo,bar'. Combine filters by filters with ';' i.e. 'tags=foo;title=bar'")
	ListCmd.PersistentFlags().StringP("filterMode", "", "exclusive", "Filter mode used by the filter flag. Default is exclusive meaning only results matching all filters will be returned. Use 'inclusive' to return all results matching one or more filters")
}
