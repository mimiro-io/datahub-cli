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
	"os"

	"github.com/mimiro-io/datahub-cli/internal/namespaces"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var NamespaceCmd = &cobra.Command{
	Use: "namespace",

	Short: "Work with Namespaces on the cli",
	Long: `Examples:
mim namespace ls
mim namespace list
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
	NamespaceCmd.AddCommand(namespaces.ListCmd)

	NamespaceCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		pterm.Println()
		result := utils.RenderMarkdown(command, "resources/doc-namespace.md")
		pterm.Println(result)
	})
}
