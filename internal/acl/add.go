package acl

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
	Short: "Add acl for client",
	Long: `Add acl for client with id. For example:
mim acl add <clientid> -file acls.json
or
mim acl add <clientid> -f acls.json
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

		acls, err := utils.ReadInput(file)
		utils.HandleError(err)
		pterm.Success.Println("Read acl file")

		sm := api.NewSecurityManager(server, token)
		err = sm.AddClientAcl(clientId, acls)

		utils.HandleError(err)

		pterm.Success.Println("Set acl for client on server")

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
	AddCmd.Flags().StringP("file", "f", "", "The public key file for this client")
}
