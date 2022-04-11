package client

import (
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var DeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete client with key",
	Long: `Delete client and related public key. For example:
mim client delete <clientid>
`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		clientId := args[0]

		sm := api.NewSecurityManager(server, token)
		err = sm.DeleteClient(clientId)

		utils.HandleError(err)

		pterm.Success.Println("Removed client from server")

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
}
