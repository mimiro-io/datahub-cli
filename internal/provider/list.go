package provider

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all token providers",
	Long: `List all token providers. For example:
mim provider list

`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DisableOutput()

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing token providers on " + server)

		sm := api.NewSecurityManager(server, token)

		tokenProviders, err := sm.ListTokenProviders()
		utils.HandleError(err)

		out, err := json.Marshal(tokenProviders)
		utils.HandleError(err)
		fmt.Println(string(out))
	},

	TraverseChildren: true,
}
