package provider

import (
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// addCmd represents the add command
var AddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add token provider",
	Long: `Add token provider. For example:
mim provider add -file provider.json
or
mim provider add -f provider.json
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

		providerConfig, err := utils.ReadInput(file)
		print(string(providerConfig))
		utils.HandleError(err)
		pterm.Success.Println("Read provider config file")

		sm := api.NewSecurityManager(server, token)
		err = sm.AddTokenProvider(providerConfig)

		utils.HandleError(err)

		pterm.Success.Println("Set token provider config")

		pterm.Println()
	},
	TraverseChildren: true,
}

func init() {
	AddCmd.Flags().StringP("file", "f", "", "The token provider json")
}
