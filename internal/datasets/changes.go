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

package datasets

import (
	"context"
	"fmt"
	"github.com/mimiro-io/datahub-cli/pkg/api"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
)

var ChangesCmd = &cobra.Command{
	Use:   "changes",
	Short: "Shows the changes for a dataset",
	Long: `Lists the changes for a dataset. For example:
mim dataset changes --dataset=mim.Cows

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format != "term" { // turn of pterm output
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		since, err := cmd.Flags().GetString("since")
		utils.HandleError(err)
		dataset, err := cmd.Flags().GetString("name")
		utils.HandleError(err)

		if dataset == "" && len(args) > 0 {
			dataset = args[0]
		}

		limit, err := cmd.Flags().GetInt("limit")
		utils.HandleError(err)

		reverse, err := cmd.Flags().GetBool("reverse")
		utils.HandleError(err)

		expanded, err := cmd.Flags().GetBool("expanded")
		utils.HandleError(err)

		pterm.DefaultSection.Println("Listing changes from " + server + fmt.Sprintf("/datasets/%s/changes", dataset))

		em := api.NewEntityManager(server, token, context.Background(), api.Changes)
		s := outputSink(format)
		if expanded {
			s = &api.SinkExpander{Sink: s}
		}
		err = em.Read(dataset, since, SaneLimit(format, limit), reverse, s)
		utils.HandleError(err)
	},
	TraverseChildren: true,
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return api.GetDatasetsCompletion(toComplete), cobra.ShellCompDirectiveNoFileComp
	},
}

func init() {
	ChangesCmd.Flags().StringP("name", "n", "", "The dataset to list changes from")
	ChangesCmd.Flags().Int("limit", 10, "Limits the number of changes to list")
	ChangesCmd.Flags().StringP("format", "f", "term", "The output format. Valid options are: term|pretty|raw")
	ChangesCmd.Flags().StringP("since", "s", "", "Send a since token to the server")
	ChangesCmd.Flags().BoolP("reverse", "r", false, "List dataset changes in reverse order: last change first")
	ChangesCmd.Flags().BoolP("expanded", "e", false, "Expand namespace prefixes in entities to full namespace URIs")
}
