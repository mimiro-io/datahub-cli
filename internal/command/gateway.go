package command

import (
	"github.com/mimiro-io/datahub-cli/internal/gateway"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var GatewayCmd = &cobra.Command{
	Use:     "gateway",
	Aliases: []string{"gway"},
	Short:   "Serve a UI gateway",
	Long: `Serves a UI gateway to allow for visual navigation of a connected data hub.
Examples:
mim gateway --login local --port 7042
`,
	Run: func(cmd *cobra.Command, args []string) {
		alias, err := cmd.Flags().GetString("login")
		port, err := cmd.Flags().GetString("port")

		if port == "" {
			port = "7042"
		}
		utils.HandleError(err)

		pterm.Success.Println("Starting gateway on http://localhost:" + port + " with " + alias + " login alias")
		gateway.StartGateway(alias, port)
	},
}

func init() {
	GatewayCmd.Flags().StringP("login", "l", "", "The account alias to use to indicate which datahub to connect to.")
	GatewayCmd.Flags().StringP("port", "p", "", "The port on which the service is exposed.")
}
