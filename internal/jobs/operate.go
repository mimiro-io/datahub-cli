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
	"context"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/pkg/api"
	"github.com/rotisserie/eris"
	"golang.org/x/sync/errgroup"
	"os"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

func CmdOperate() *cobra.Command {

	var (
		operation string
		ids       []string
		since     string
		jobType   string
		wait      bool
	)

	cmd := &cobra.Command{
		Use:   "operate",
		Short: "Run, stop, pause, resume and kill jobs with given job id",
		Long: `Run, stop, pause, resume and kill jobs with given job id, For example:

	mim jobs operate -i <id> -o stop
	mim jobs operate --id <id> --operation stop
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(ids) == 0 && len(args) > 0 {
				ids = []string{args[0]}
			}

			if len(ids) == 0 {
				_ = cmd.Usage()
				os.Exit(0)
			}

			// we only allow 1 job if we are following, because of reasons of complexity
			if len(ids) > 1 && operation == "run" && wait {
				utils.HandleError(eris.New("When following a job, only 1 id is allowed"))
			}

			server, token, err := login.ResolveCredentials()
			utils.HandleError(err)

			pterm.EnableDebugMessages()

			pterm.DefaultSection.Printf("Executing operation '%s' on %s", pterm.Cyan(operation), server)
			pterm.Println()

			jm := api.NewJobManager(server, token)
			resolvedIds := jm.ResolveIds(ids...)

			// if the operation is kill, ask for confirmation before proceeding
			if operation == "kill" {
				pterm.DefaultSection.Printf("Do you really want to kill job(s), type (y)es or (n)o and then press enter:")
				if !utils.AskForConfirmation() {
					pterm.Warning.Println("Aborted kill!")
					os.Exit(0)
				}
			} else if operation == "run" && wait {
				// if operation is run, and we are following, handle differently for now
				runAndWait(jm, resolvedIds[0], jobType)
				os.Exit(0)
			}

			g, _ := errgroup.WithContext(context.Background())
			g.SetLimit(10) // not sure if needed, but this will limit to 10 requests at once
			p, _ := pterm.DefaultProgressbar.WithTotal(len(resolvedIds)).WithTitle("Executing operation" + operation).Start()
			for _, rid := range resolvedIds {
				id := rid
				g.Go(func() error {
					err2 := operate(jm, operation, id, since, jobType)
					if err2 == nil {
						pterm.Success.Println("Processed " + id.Title)
					} else {
						pterm.Error.Printf("Processed %s, error: %s \n", id.Title, pterm.Red(err2.Error()))
					}
					p.Increment()
					return err2
				})

			}
			err = g.Wait()
			if err != nil {
				pterm.Println()
				pterm.Println("There was an error with 1 or more operations:")
				pterm.Error.Println(err)
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
	cmd.Flags().StringVarP(&operation, "operation", "o", "", "The name of the operation")
	cmd.Flags().StringSliceVarP(&ids, "id", "i", []string{}, "The job id or name of the job you want to operate on. Can be multiple ids.")
	cmd.Flags().StringVarP(&since, "since", "s", "", "The since token to reset to, if resetting or running")
	cmd.Flags().StringVarP(&jobType, "jobType", "t", "", "Job type for operation run: fullsync or incremental")
	cmd.Flags().BoolVarP(&wait, "wait", "w", false, "Use together with run operation to wait for the job to finish. When set you can only have 1 job id.")
	_ = cmd.RegisterFlagCompletionFunc("operation", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"run", "stop", "pause", "resume", "kill", "reset"}, cobra.ShellCompDirectiveDefault
	})
	_ = cmd.RegisterFlagCompletionFunc("jobType", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"fullsync", "incremental"}, cobra.ShellCompDirectiveDefault
	})

	return cmd
}

// operate organizes the different Operate calls depending on user input.
func operate(jm *api.JobManager, operation string, id api.JobId, since string, jobType string) error {
	ctx := context.Background()
	var err error
	switch operation {
	case "pause":
		_, err = jm.Operate.Pause(ctx, id.Id)
	case "resume":
		_, err = jm.Operate.Resume(ctx, id.Id)
	case "reset":
		_, err = jm.Operate.Reset(ctx, id.Id, since)
	case "kill":
		_, err = jm.Operate.Kill(ctx, id.Id)
	case "run":
		running, err2 := jm.GetJobStatus(id.Id)
		if err2 != nil {
			return err2
		}
		if len(running) == 0 { // not running
			_, err = jm.Operate.Run(ctx, id.Id, jobType)
		}
	case "test":
		_, err = jm.Operate.Test(ctx, id.Id)
	default:
		pterm.Warning.Println("Unsupported operation! Supported is run, pause, resume, reset and kill.")
	}
	return err
}

// runAndWait is called when wait flag is set and the operation is "run". For now this has to be handled
// differently than the others, as the ui component is quite different. We also only allow to run 1 job at a
// time when the wait flag is set.
func runAndWait(jm *api.JobManager, id api.JobId, jobType string) {
	// is job running?
	running, err := jm.GetJobStatus(id.Id)
	utils.HandleError(err)
	runningId := ""
	if len(running) == 0 { // not running
		job, err := jm.Operate.Run(context.Background(), id.Id, jobType)
		utils.HandleError(err)

		runningId = job.JobId

		pterm.Success.Println("Job was started")
	} else {
		pterm.Success.Println("Job is already running, reattaching")
		runningId = running[0].JobId
	}

	pterm.Println()
	spinner, err := pterm.DefaultSpinner.Start(fmt.Sprintf("Processing job with id '%s'", runningId))
	utils.HandleError(err)
	err = followJob(runningId, *jm)
	if err != nil {
		spinner.Fail("Failed following job")
		utils.HandleError(err) // this should be an error in the handling only, not in the job running itself
	}
	// we can fetch the job history
	history, err := jm.GetJobHistoryForId(runningId)
	if err != nil {
		spinner.Fail(err)
	} else {
		if history.LastError == "" {
			spinner.Success("Finished")
		} else {
			spinner.Fail("Finished with error")
		}
		pterm.Info.Println(fmt.Sprintf("Job started: 	%v", utils.Date(history.Start)))
		pterm.Info.Println(fmt.Sprintf("Job ended: 	%v", utils.Date(history.End)))
		pterm.Info.Println(fmt.Sprintf("Processed %v entities", history.Processed))
		if history.LastError != "" {
			pterm.Error.Println("Last error:")
			pterm.Println(history.LastError)
		}
	}
}

func followJob(jobId string, jm api.JobManager) error {
	for {
		status, err := jm.GetJobStatus(jobId)
		if err != nil {
			return err
		}

		if len(status) == 0 {
			break
		}

		time.Sleep(5 * time.Second)
	}
	return nil
}
