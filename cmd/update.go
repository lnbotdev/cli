package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/lnbotdev/cli/internal/update"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Check for updates or show upgrade instructions",
	Long:  `Check if a newer version of lnbot is available and show how to upgrade.`,
	Example: `  lnbot update`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Current version: %s\n", version)

		latest, available := update.CheckForUpdate(version)
		if !available {
			fmt.Println("You're up to date.")
			return
		}

		fmt.Printf("Latest version:  %s\n", latest)
		fmt.Println()
		fmt.Println("To update:")

		switch runtime.GOOS {
		case "windows":
			fmt.Println("  PowerShell: iwr -useb https://ln.bot/install.ps1 | iex")
			fmt.Println("  CMD:        curl -fsSL https://ln.bot/install.cmd -o install.cmd && install.cmd && del install.cmd")
		default:
			fmt.Println("  curl -fsSL https://ln.bot/install.sh | bash")
		}
	},
}
