package command

import (
	"github.com/mimiro-io/datahub-cli/internal/provider"
	"github.com/spf13/cobra"
	"os"
)

var ProviderCmd = &cobra.Command{
	Use:     "provider",
	Aliases: []string{"provider"},
	Short:   "Manage data hub token providers from the cli",
	Long: `See available Commands.
Examples:
	mim provider add -f config.json
	mim provider list
	mim provider delete <providerid>
`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Usage()
			os.Exit(0)
		}
	},
}

func init() {
	ProviderCmd.AddCommand(provider.AddCmd)
	ProviderCmd.AddCommand(provider.ListCmd)
}
