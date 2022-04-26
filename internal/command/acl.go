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

	"github.com/mimiro-io/datahub-cli/internal/acl"

	"github.com/spf13/cobra"
)

var AclCmd = &cobra.Command{
	Use:     "acl",
	Aliases: []string{"acl"},
	Short:   "Manage data hub client acls from the cli",
	Long: `See available Commands.
Examples:
	mim acl set <clientid> -f acls.json
	mim acl get <clientid>
	mim acl delete <clientid>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	AclCmd.AddCommand(acl.AddCmd)
	AclCmd.AddCommand(acl.GetCmd)
}
