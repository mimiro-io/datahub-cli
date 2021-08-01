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

package txns

import (
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var ExecuteCmd = &cobra.Command{
	Use:   "execute",
	Short: "Execute transaction",
	Long: `Executes a transaction based on data provided. For example:
mim txn execute -file <txn.json>
or
mim txn execute -f <txn.json>
`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		file, err := cmd.Flags().GetString("file")
		utils.HandleError(err)

		txnData, err := utils.ReadInput(file)
		utils.HandleError(err)
		pterm.Success.Println("Read transaction data")

		txnManager := api.NewTxnManager(server, token)
		err = txnManager.ExecuteTransaction(txnData)

		utils.HandleError(err)

		pterm.Success.Println("Transaction executed")

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
	ExecuteCmd.Flags().StringP("file", "f", "", "The transaction data file to execute")
}

