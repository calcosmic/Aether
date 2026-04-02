package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `To load completions:

Bash:
  $ source <(aether completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ aether completion bash > /etc/bash_completion.d/aether
  # macOS:
  $ aether completion bash > $(brew --prefix)/etc/bash_completion.d/aether

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ aether completion zsh > "${fpath[1]}/_aether"

  # You will need to start a new shell for this setup to take effect.

Fish:
  $ aether completion fish | source

  # To load completions for each session, execute once:
  $ aether completion fish > ~/.config/fish/completions/aether.fish

PowerShell:
  PS> aether completion powershell | Out-String | Invoke-Expression

  # To load completions for each session, execute once:
  PS> aether completion powershell > aether.ps1
  # and source this file from your PowerShell profile.
`,
	Args:              cobra.ExactValidArgs(1),
	ValidArgs:         []string{"bash", "zsh", "fish", "powershell"},
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletionV2(os.Stdout, true)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		default:
			return nil
		}
	},
}
