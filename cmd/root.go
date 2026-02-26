package cmd

import (
	"errors"
	"fmt"
	"os"
	"text/template"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
)

var (
	walletFlag string
	jsonFlag   bool
	yesFlag    bool

	cfg *config.Config
)

const version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:   "lnbot",
	Short: "Lightning wallets for agents",
	Long:  `ln.bot — Bitcoin Lightning wallets for AI agents`,
	Example: `  $ lnbot init
  $ lnbot wallet create --name agent01
  $ lnbot invoice create --amount 1000 --memo "coffee"
  $ lnbot pay alice@ln.bot --amount 500
  $ lnbot balance`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		var err error
		cfg, err = config.Load()
		return err
	},
}

const rootHelpTmpl = `{{.Long}}

{{- if .HasAvailableSubCommands}}
{{- range .Groups}}

{{.Title}}
  {{- range (index $.Commands .ID)}}
  {{rpad .Name .NamePadding}} {{.Short}}
  {{- end}}
{{- end}}

{{- if .HasAvailablePersistentFlags}}

Flags:
{{.PersistentFlags.FlagUsages}}
{{- end}}

{{- end}}

{{- if .HasExample}}

Examples:
{{.Example}}
{{- end}}

Docs: https://ln.bot/docs
`

const leafHelpTmpl = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .Runnable}}Usage:
  {{.UseLine}}

{{end}}{{if .HasAvailableLocalFlags}}Flags:
{{.LocalFlags.FlagUsages}}
{{end}}{{if .HasAvailableInheritedFlags}}Global Flags:
{{.InheritedFlags.FlagUsages}}
{{end}}{{if .HasExample}}Examples:
{{.Example}}
{{end}}`

const groupHelpTmpl = `{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}

{{end}}{{if .HasAvailableSubCommands}}Commands:
{{- range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding}} {{.Short}}
{{- end}}{{end}}

{{end}}{{if .HasAvailableLocalFlags}}Flags:
{{.LocalFlags.FlagUsages}}
{{end}}{{if .HasAvailableInheritedFlags}}Global Flags:
{{.InheritedFlags.FlagUsages}}
{{end}}{{if .HasExample}}Examples:
{{.Example}}

{{end}}Use "{{.CommandPath}} [command] --help" for more information about a command.
`

func init() {
	rootCmd.PersistentFlags().StringVarP(&walletFlag, "wallet", "w", "", "target a specific wallet")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "output as JSON")
	rootCmd.PersistentFlags().BoolVarP(&yesFlag, "yes", "y", false, "skip confirmation prompts")

	rootCmd.AddGroup(
		&cobra.Group{ID: "start", Title: "Getting Started:"},
		&cobra.Group{ID: "money", Title: "Money:"},
		&cobra.Group{ID: "identity", Title: "Identity:"},
		&cobra.Group{ID: "security", Title: "Security:"},
		&cobra.Group{ID: "integrations", Title: "Integrations:"},
		&cobra.Group{ID: "other", Title: "Additional:"},
	)

	initCmd.GroupID = "start"
	walletCmd.GroupID = "start"

	balanceCmd.GroupID = "money"
	invoiceCmd.GroupID = "money"
	payCmd.GroupID = "money"
	paymentCmd.GroupID = "money"
	transactionsCmd.GroupID = "money"

	addressCmd.GroupID = "identity"
	whoamiCmd.GroupID = "identity"
	statusCmd.GroupID = "identity"

	keyCmd.GroupID = "security"
	backupCmd.GroupID = "security"
	restoreCmd.GroupID = "security"

	webhookCmd.GroupID = "integrations"
	mcpCmd.GroupID = "integrations"

	completionCmd.GroupID = "other"
	versionCmd.GroupID = "other"

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(walletCmd)
	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(whoamiCmd)
	rootCmd.AddCommand(invoiceCmd)
	rootCmd.AddCommand(payCmd)
	rootCmd.AddCommand(paymentCmd)
	rootCmd.AddCommand(transactionsCmd)
	rootCmd.AddCommand(keyCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
	rootCmd.AddCommand(addressCmd)
	rootCmd.AddCommand(webhookCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(versionCmd)

	cobra.AddTemplateFuncs(template.FuncMap{
		"index": func(cmds []*cobra.Command, groupID string) []*cobra.Command {
			var result []*cobra.Command
			for _, c := range cmds {
				if c.GroupID == groupID && c.IsAvailableCommand() {
					result = append(result, c)
				}
			}
			return result
		},
	})

	rootCmd.SetHelpTemplate(rootHelpTmpl)

	for _, cmd := range []*cobra.Command{walletCmd, invoiceCmd, paymentCmd, addressCmd, keyCmd, backupCmd, restoreCmd, webhookCmd, mcpCmd} {
		cmd.SetHelpTemplate(groupHelpTmpl)
	}

	// Leaf commands: use default cobra template (don't inherit root's grouped template)
	leafCmds := []*cobra.Command{
		initCmd, balanceCmd, statusCmd, whoamiCmd, payCmd, transactionsCmd,
		completionCmd, versionCmd,
	}
	for _, cmd := range leafCmds {
		cmd.SetHelpTemplate(leafHelpTmpl)
	}
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Built-in commands
// ---------------------------------------------------------------------------

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print lnbot version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("lnbot %s\n", version)
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create local config file",
	Long: `Initialize the lnbot CLI by creating a config file.

The config is stored at ~/.config/lnbot/config.json (override with
LNBOT_CONFIG env var). Run this once, then create your first wallet.`,
	Example: `  lnbot init
  lnbot wallet create --name agent01`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil {
			fmt.Println("Config already exists at", config.Path())
			return nil
		}
		var err error
		cfg, err = config.Init()
		if err != nil {
			return err
		}
		printSuccess("Config created at " + config.Path())
		fmt.Println()
		fmt.Println("  Next: lnbot wallet create --name <name>")
		return nil
	},
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func requireConfig() error {
	if cfg == nil {
		return fmt.Errorf("no config found — run 'lnbot init' first")
	}
	return nil
}

func confirm(prompt string) bool {
	if yesFlag {
		return true
	}
	fmt.Printf("%s (y/N) ", prompt)
	var response string
	fmt.Scanln(&response)
	return response == "y" || response == "Y"
}

func printSuccess(msg string) {
	fmt.Printf("✓ %s\n", msg)
}

func printWarning(msg string) {
	fmt.Printf("⚠ %s\n", msg)
}

func truncateKey(key string) string {
	if len(key) <= 16 {
		return key
	}
	return key[:12] + "..." + key[len(key)-4:]
}

func apiError(action string, err error) error {
	var apiErr *lnbot.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: %s", action, apiErr.Message)
	}
	return fmt.Errorf("%s: %w", action, err)
}
