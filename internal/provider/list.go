package provider

import (
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-cli/internal/api"
	"github.com/mimiro-io/datahub-cli/internal/login"
	"github.com/mimiro-io/datahub-cli/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
)

var ListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all token providers",
	Long: `List all token providers. For example:
mim provider list

`,
	Run: func(cmd *cobra.Command, args []string) {
		format := utils.ResolveFormat(cmd)
		if format == "json" {
			pterm.DisableOutput()
		}

		server, token, err := login.ResolveCredentials()
		utils.HandleError(err)

		pterm.EnableDebugMessages()

		pterm.DefaultSection.Println("Listing token providers on " + server)

		sm := api.NewSecurityManager(server, token)

		tokenProviders, err := sm.ListTokenProviders()
		utils.HandleError(err)

		printOutput(tokenProviders, format)
	},

	TraverseChildren: true,
}

func printOutput(output []api.ProviderConfig, format string) {

	jd, err := json.Marshal(output)
	utils.HandleError(err)

	switch format {
	case "json":
		fmt.Println(string(jd))
	case "pretty":
		p := pretty.Pretty(jd)
		result := pretty.Color(p, nil)
		fmt.Println(string(result))
	default:
		fmt.Println(string(jd))
	}
}
