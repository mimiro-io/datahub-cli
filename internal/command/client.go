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
	"github.com/mimiro-io/datahub-cli/internal/client"
	"os"

	"github.com/spf13/cobra"
)

var ClientCmd = &cobra.Command{
	Use:     "client",
	Aliases: []string{"client"},
	Short:   "Manage data hub clients from the cli",
	Long: `See available Commands.
Examples:
	min client add <clientid> -f key.pub
	mim client delete <clientid>
	mim client list
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	ClientCmd.AddCommand(client.AddCmd)
	ClientCmd.AddCommand(client.ListCmd)
	ClientCmd.AddCommand(client.DeleteCmd)
}
