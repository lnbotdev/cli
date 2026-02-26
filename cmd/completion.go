package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion <bash|zsh|fish|powershell>",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for lnbot.

To load completions:

Bash:
  source <(lnbot completion bash)

  # Persist across sessions (Linux):
  lnbot completion bash > /etc/bash_completion.d/lnbot

  # Persist across sessions (macOS):
  lnbot completion bash > $(brew --prefix)/etc/bash_completion.d/lnbot

Zsh:
  source <(lnbot completion zsh)

  # Persist across sessions:
  lnbot completion zsh > "${fpath[1]}/_lnbot"

Fish:
  lnbot completion fish | source

  # Persist across sessions:
  lnbot completion fish > ~/.config/fish/completions/lnbot.fish

PowerShell:
  lnbot completion powershell | Out-String | Invoke-Expression`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
