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

package transform

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/web"
	"os"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// ImportCmd allows to import a *.js (or a *.ts) file into an existing job
var ImportCmd = &cobra.Command{
	Use:   "import",
	Short: "Import a transform file into a job",
	Long: `Import a transform file into a job. For example:
mim transform import --job-id <my-job> --file <transform.js>
or
mim transform import <my-job> -f <transform.js>

`,
	Run: func(cmd *cobra.Command, args []string) {

		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		file, err := cmd.Flags().GetString("file")
		utils.HandleError(err)

		introSpinner, err := pterm.DefaultSpinner.WithRemoveWhenDone(false).Start("Compiling script")
		utils.HandleError(err)

		importer := NewImporter(file)
		code, err := importer.Import()
		utils.HandleError(err)

		introSpinner.Success("Done compiling and minifying")
		introSpinner.Stop()

		encode, err := cmd.Flags().GetBool("encode-only")
		jobId, err := cmd.Flags().GetString("job-id")
		if len(args) > 0 { // allow job id as first argument
			jobId = args[0]
		}

		if jobId == "" {
			encode = true
		}

		encoded := importer.Encode(code)
		if encode {
			pterm.Println(encoded)
			pterm.Println()
			os.Exit(1)
		}
		pterm.Success.Println("Done base64 encoding")

		jobManager := api.NewJobManager(server, token)

		resolvedJobId := jobManager.ResolveId(jobId)

		job, err := jobManager.GetJob(resolvedJobId)
		utils.HandleError(err)
		pterm.Success.Println("Fetched job with id " + jobId)

		tfr := make(map[string]interface{})
		tfr["Type"] = "JavascriptTransform"
		tfr["Code"] = encoded

		job.Transform = tfr

		err = updateJob(server, token, job)
		utils.HandleError(err)

		pterm.Success.Println(fmt.Sprintf("Job '%s' was updated", jobId))

		pterm.Println()
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetJobsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func updateJob(server string, token string, job *api.Job) error {
	data, err := json.Marshal(job)
	if err != nil {
		return err
	}

	_, err = web.PostRequest(server, token, "/jobs", data)
	return err
}

func getJob(server string, token string, jobId string) (*api.Job, error) {
	res, err := web.GetRequest(server, token, fmt.Sprintf("/jobs/%s", jobId))
	if err != nil {
		if strings.Index(err.Error(), "404") > -1 {
			return nil, errors.New(fmt.Sprintf("could not find job '%s' to attach to", jobId))
		}
		return nil, err
	}

	job := &api.Job{}
	err = json.Unmarshal(res, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func init() {
	ImportCmd.Flags().StringP("file", "f", "", "The file to export the transform to.")
	ImportCmd.Flags().StringP("job-id", "j", "", "The id of the job to export the transform from.")
	ImportCmd.Flags().BoolP("encode-only", "e", false, "Only base 64 encode the transform code and print it to stdout")
}
