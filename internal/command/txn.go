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
	"github.com/mimiro-io/datahub-cli/internal/txns"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// jobsCmd represents the jobs command
var TxnsCmd = &cobra.Command{
	Use:     "txn",
	Aliases: []string{"transactions"},
	Short:   "Work with transactions from the cli",
	Long: `See available Commands.
Examples:
	mim txn execute -f txn.json
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	TxnsCmd.AddCommand(txns.ExecuteCmd)
	TxnsCmd.SetHelpFunc(func(command *cobra.Command, strings []string) {
		pterm.Println()
		result := docs.RenderMarkdown(command, "doc-txns.md")
		pterm.Println(result)
	})
}

