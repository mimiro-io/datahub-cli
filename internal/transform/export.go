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
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

const export = "function transform_entities(entities)"

// ExportCmd allows to export an existing transform as javascript
var ExportCmd = &cobra.Command{
	Use:     "get",
	Aliases: []string{"export"},
	Short:   "Get an already existing transform",
	Long: `Get an already existing transform. For example:
mim transform get --id <my-job> -file out.js
or
mim transform export <my-job> > out.js

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput = true
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		jobId, _ := cmd.Flags().GetString("id")
		if jobId == "" && len(args) > 0 {
			jobId = args[0]
		}

		if jobId == "" {
			utils.HandleError(errors.New("job id is missing"))
		}

		jobManager := api.NewJobManager(server, token)
		job, err := jobManager.GetJob(jobId)
		utils.HandleError(err)

		output(job)

		pterm.Println()
	},
	TraverseChildren: true,
}

func output(job *api.Job) {
	if job.Transform != nil {
		if transform, ok := job.Transform["Code"]; ok {
			out, err := base64.StdEncoding.DecodeString(transform.(string))
			utils.HandleError(err)

			// we should readd the export function part
			js := string(out)
			index := strings.Index(js, export)
			if index > -1 {
				js = js[:index] + "export " + js[index:]
			}

			fmt.Println(js)
		}
	}

}

func init() {
	ExportCmd.Flags().StringP("file", "f", "", "The file to export the transform to.")
	ExportCmd.Flags().StringP("id", "j", "", "The id of the job to export the transform from.")
}
