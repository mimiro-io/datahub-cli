package client

import (
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/mimiro-io/datahub-cli/pkg/api"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add client with key",
	Long: `Add client and related public key. For example:
mim client add <clientid> -file <key.pub>
or
mim client add <clientid> -f <key.pub>
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

		file, err := cmd.Flags().GetString("file")
		utils.HandleError(err)

		publicKey, err := utils.ReadInput(file)
		utils.HandleError(err)
		pterm.Success.Println("Read public key file")

		sm := api.NewSecurityManager(server, token)
		err = sm.AddClient(clientId, publicKey)

		utils.HandleError(err)

		pterm.Success.Println("Added client to server")

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
	AddCmd.Flags().StringP("file", "f", "", "The public key file for this client")
}
