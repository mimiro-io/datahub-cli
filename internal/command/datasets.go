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

	"github.com/mimiro-io/datahub-cli/internal/datasets"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var DatasetCmd = &cobra.Command{
	Use:     "dataset",
	Aliases: []string{"datasets"},
	Short:   "Manage datahub datasets from the cli",
	Long: `Manage datahub datasets from cli such as add, delete, describe and so on. See available Commands.
Examples:
mim dataset list
mim dataset create
mim dataset delete
mim dataset entities --name=<dataset>
mim dataset changes --name=<dataset>
mim dataset store --name=<dataset> --filename=<entities file to load>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
	TraverseChildren: true,
}

func init() {
	DatasetCmd.AddCommand(datasets.ListCmd)
	DatasetCmd.AddCommand(datasets.EntitiesCmd)
	DatasetCmd.AddCommand(datasets.ChangesCmd)
	DatasetCmd.AddCommand(datasets.DeleteCmd)
	DatasetCmd.AddCommand(datasets.CreateCmd)
	DatasetCmd.AddCommand(datasets.GetCmd)
	DatasetCmd.AddCommand(datasets.StoreCmd)
	DatasetCmd.AddCommand(datasets.RenameCmd)

	DatasetCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		pterm.Println()
		result := docs.RenderMarkdown(command, "doc-dataset.md")
		pterm.Println(result)
	})
}
