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
	"github.com/spf13/cobra"
	"os"
)

var CompletionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script",
	Long: `To load completions:

Bash:

  $ source <(mim completion bash)

  # To load completions for each session, execute once:

  # Linux:
  $ mim completion bash > /etc/bash_completion.d/mim

  # macOS:
  $ mim completion bash > /usr/local/etc/bash_completion.d/mim
  # Mac OS comes with a really old bash version. So in order to
  # get bash completion to work properly, you need to update it:

  $ brew install bash bash-completion
  # Add the new bash shell to the list of available shells:
  $ sudo vim /etc/shells
  # Append /usr/local/bin/bash to the list.
  # Change default shell by running
  $ chsh -s /usr/local/bin/bash
  # If you wish to change default shell for root, run the same
  # command with sudo.

  # Finally to source completions in every session, you need to add
  # this to your ~/.bash_profile
  $ echo "[ -f /usr/local/etc/bash_completion ] && . /usr/local/etc/bash_completion" >> ~/.bash_profile



Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ mim completion zsh > "${fpath[1]}/_mim"

  # You will need to start a new shell for this setup to take effect.

fish:

  $ mim completion fish | source

  # To load completions for each session, execute once:
  $ mim completion fish > ~/.config/fish/completions/mim.fish

PowerShell:

  PS> mim completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> mim completion powershell > mim.ps1
  # and source this file from your PowerShell profile.
`,
	DisableFlagsInUseLine: true,
	Hidden:                true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletion(os.Stdout)
		}
	},
}
