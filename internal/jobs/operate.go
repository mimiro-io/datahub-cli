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
	"github.com/mimiro-io/datahub-cli/internal/web"
	"os"
	"time"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// OperateCmd represents the operate command
var OperateCmd = &cobra.Command{
	Use:   "operate",
	Short: "Run, stop, pause, resume and kill jobs with given jobid",
	Long: `Run, stop, pause, resume and kill jobs with given jobid, For example:
mim jobs operate -i <jobid> -o stop
or
mim jobs operate --id <jobid> --operation stop

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		idOrTitle, err := cmd.Flags().GetString("id")
		utils.HandleError(err)
		if idOrTitle == "" && len(args) > 0 {
			idOrTitle = args[0]
		}

		operation, err := cmd.Flags().GetString("operation")
		utils.HandleError(err)

		since, err := cmd.Flags().GetString("since")
		utils.HandleError(err)

		jm := api.NewJobManager(server, token)
		id := jm.ResolveId(idOrTitle)

		pterm.DefaultSection.Printf("Execute operation " + operation + " on job with id: " + id + " (" + idOrTitle + ") on " + server)
		pterm.Println()
		if operation == "run" {
			// is job running?
			running, err := jm.GetJobStatus(id)
			utils.HandleError(err)
			runningId := ""
			if len(running) == 0 { // not running
				jobType, err := cmd.Flags().GetString("jobType")
				utils.HandleError(err)

				endpoint := fmt.Sprintf("/job/%s/run", id)
				if jobType != "" {
					endpoint = endpoint + "?jobType=" + jobType
				}
				response, err := web.PutRequest(server, token, endpoint)
				utils.HandleError(err)
				job, err := getJobResponse(response)
				utils.HandleError(err)

				runningId = job.JobId

				pterm.Success.Println("Job was started")
			} else {
				pterm.Success.Println("Job is already running, reattaching")
				runningId = running[0].JobId
			}

			// follow the job
			err = followJob(runningId, *jm)
			utils.HandleError(err)
		} else if operation == "pause" {
			_, err = web.PutRequest(server, token, fmt.Sprintf("/job/%s/pause", id))
			utils.HandleError(err)

			pterm.Success.Println("Job was paused")
		} else if operation == "resume" {
			_, err = web.PutRequest(server, token, fmt.Sprintf("/job/%s/resume", id))
			utils.HandleError(err)

			pterm.Success.Println("Job was resumed")
		} else if operation == "reset" {
			_, err = resetJob(server, token, id, since)
			utils.HandleError(err)

			pterm.Success.Println("Job was reset")
		} else if operation == "kill" {
			pterm.DefaultSection.Printf("Do you really want to kill job, type (y)es or (n)o and then press enter:")
			if utils.AskForConfirmation() {
				_, err = web.PutRequest(server, token, fmt.Sprintf("/job/%s/kill", id))
				utils.HandleError(err)

				pterm.Success.Println("Job was killed")
				pterm.Println()
			} else {
				pterm.Warning.Println("Aborted kill!")
			}

		} else {
			pterm.Warning.Println("Unsupported operation! Supported is run, pause, resume, reset and kill.")
		}

		pterm.Println()

	},
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

type jobResponse struct {
	JobId string `json:"jobId"`
}

func getJobResponse(response []byte) (*jobResponse, error) {
	// parse the reponse to get the id
	job := &jobResponse{}
	err := json.Unmarshal(response, job)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func followJob(jobId string, jm api.JobManager) error {
	pterm.Println()
	spinner, err := pterm.DefaultSpinner.Start(fmt.Sprintf("Processing job with id '%s'", jobId))
	utils.HandleError(err)
	success := true
	msg := "Finished"
	for {
		status, err := jm.GetJobStatus(jobId)
		if err != nil {
			success = false
			msg = err.Error()
			break
		}

		if len(status) == 0 {
			break
		}

		time.Sleep(5 * time.Second)
	}
	if success {
		spinner.Success(msg)
	} else {
		spinner.Fail(msg)
	}
	return nil
}

func resetJob(server string, token string, jobId string, since string) ([]byte, error) {
	endpoint := fmt.Sprintf("/job/%s/reset", jobId)
	if since != "" {
		endpoint = "?since=" + since
	}

	return web.PutRequest(server, token, endpoint)
}

func init() {
	OperateCmd.Flags().StringP("operation", "o", "", "The name of the operation")
	OperateCmd.Flags().StringP("id", "i", "", "The jobid name of the job you want to get operate on")
	OperateCmd.Flags().StringP("since", "s", "", "The since token to reset to, if resetting or running")
	OperateCmd.Flags().StringP("jobType", "t", "", "jobType for operation run: fullsync or incremental")
	OperateCmd.RegisterFlagCompletionFunc("operation", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"run", "stop", "pause", "resume", "kill", "reset"}, cobra.ShellCompDirectiveDefault
	})
	OperateCmd.RegisterFlagCompletionFunc("jobType", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"fullsync", "incremental"}, cobra.ShellCompDirectiveDefault
	})
}
