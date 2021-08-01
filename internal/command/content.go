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

	"github.com/mimiro-io/datahub-cli/internal/content"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var ContentCmd = &cobra.Command{
	Use:     "content",
	Aliases: []string{"contents"},
	Short:   "Manage datahub content from the cli",
	Long: `Manage datahub content from cli such as add, delete, describe and so on. See available Commands.
Examples:
mim content list
mim content add
mim content delete
mim content show
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
	ContentCmd.AddCommand(content.ListCmd)
	ContentCmd.AddCommand(content.ShowCmd)
	ContentCmd.AddCommand(content.AddCmd)
	ContentCmd.AddCommand(content.DeleteCmd)

	ContentCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {

		pterm.Println()
		result := docs.RenderMarkdown(command, "doc-content.md")
		pterm.Println(result)
	})

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// jobsCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	//LoginCmd.Flags().StringP("server", "s", "https://api.mimiro.io", "The server to login against. This will set the server permanently until the next login")
	//LoginCmd.Flags().String("existing-token", "", "Store an existing token for use")
}
