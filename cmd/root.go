package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
	"github.com/lnbotdev/cli/internal/update"
)

var (
	walletFlag string
	jsonFlag   bool
	yesFlag    bool

	cfg *config.Config
)

const version = "1.0.0"

var rootCmd = &cobra.Command{
	Use:   "lnbot",
	Short: "Lightning wallets for agents",
	Long:  `ln.bot — Bitcoin Lightning wallets for AI agents`,
	Example: `  $ lnbot init
  $ lnbot wallet create
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
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if latest, ok := update.CheckForUpdate(version); ok {
			fmt.Fprintf(os.Stderr, "\nUpdate available: %s → %s\n", version, latest)
			fmt.Fprintf(os.Stderr, "Run: curl -fsSL https://ln.bot/install.sh | bash\n")
		}
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
	rootCmd.PersistentFlags().StringVarP(&walletFlag, "wallet", "w", "", "wallet ID or name (default: active wallet)")
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

	updateCmd.GroupID = "other"
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
	rootCmd.AddCommand(updateCmd)
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

	leafCmds := []*cobra.Command{
		initCmd, balanceCmd, statusCmd, whoamiCmd, payCmd, transactionsCmd,
		updateCmd, completionCmd, versionCmd,
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
	Short: "Create a new account and wallet",
	Long: `Register a new ln.bot account, create your first wallet, and save
credentials locally.

The config is stored at ~/.config/lnbot/config.json (override with
LNBOT_CONFIG env var). Run this once — subsequent wallets are created
with 'lnbot wallet create'.`,
	Example: `  lnbot init`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg != nil {
			fmt.Println("Already initialized. Config at", config.Path())
			fmt.Println()
			fmt.Println("  To create another wallet: lnbot wallet create")
			return nil
		}

		fmt.Print("Registering account... ")

		ctx := context.Background()
		ln := config.AnonClient()
		account, err := ln.Register(ctx)
		if err != nil {
			fmt.Println()
			return apiError("registering account", err)
		}
		fmt.Println("done")

		fmt.Print("Creating wallet... ")
		authed := lnbot.New(account.PrimaryKey)
		wallet, err := authed.Wallets.Create(ctx)
		if err != nil {
			fmt.Println()
			return apiError("creating wallet", err)
		}
		fmt.Println("done")

		cfg, err = config.Init(account.PrimaryKey, account.SecondaryKey, wallet.WalletID)
		if err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"primary_key":         account.PrimaryKey,
				"secondary_key":       account.SecondaryKey,
				"wallet_id":           wallet.WalletID,
				"wallet_name":         wallet.Name,
				"address":             wallet.Address,
				"recovery_passphrase": account.RecoveryPassphrase,
			})
		}

		fmt.Println()
		printSuccess("Account and wallet created")
		fmt.Printf("  wallet:   %s (%s)\n", wallet.Name, wallet.WalletID)
		fmt.Printf("  address:  %s\n", wallet.Address)
		fmt.Printf("  api key:  %s\n", truncateKey(account.PrimaryKey))
		fmt.Println()
		printWarning("Recovery passphrase (save this — shown only once):")
		fmt.Printf("  %s\n", account.RecoveryPassphrase)
		fmt.Println()
		fmt.Printf("  Config saved to %s\n", config.Path())
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

// resolveWalletID returns the wallet ID to use, from the --wallet flag or active config.
// If the flag looks like a wallet ID (wal_...) it is used directly.
// Otherwise it is treated as a wallet name and resolved via the API.
func resolveWalletID() (string, error) {
	if err := requireConfig(); err != nil {
		return "", err
	}
	if walletFlag == "" {
		if cfg.ActiveWalletID == "" {
			return "", fmt.Errorf("no active wallet — run 'lnbot wallet use <id>'")
		}
		return cfg.ActiveWalletID, nil
	}
	if strings.HasPrefix(walletFlag, "wal_") {
		return walletFlag, nil
	}
	// Look up by name from API
	wallets, err := cfg.Client().Wallets.List(context.Background())
	if err != nil {
		return "", fmt.Errorf("looking up wallet: %w", err)
	}
	for _, w := range wallets {
		if w.Name == walletFlag {
			return w.WalletID, nil
		}
	}
	return "", fmt.Errorf("wallet %q not found", walletFlag)
}

// resolveWallet returns a WalletHandle for the active or --wallet-specified wallet.
func resolveWallet() (*lnbot.WalletHandle, error) {
	id, err := resolveWalletID()
	if err != nil {
		return nil, err
	}
	return cfg.Client().Wallet(id), nil
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
