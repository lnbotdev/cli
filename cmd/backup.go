package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup <command>",
	Short: "Back up wallet credentials",
	Long: `Create a backup so you can recover your wallet later.

Two methods are available:
  recovery  — generates a 12-word passphrase you can store offline
  passkey   — registers a WebAuthn passkey (browser only)`,
}

func init() {
	backupCmd.AddCommand(backupPasskeyCmd)
	backupCmd.AddCommand(backupRecoveryCmd)
}

var backupPasskeyCmd = &cobra.Command{
	Use:   "passkey",
	Short: "Register a passkey (browser only)",
	Long: `Register a WebAuthn passkey for wallet recovery.

This requires a browser with WebAuthn support and is not available in
the CLI. Use the web terminal at https://ln.bot instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Passkey registration requires a browser with WebAuthn support.")
		fmt.Fprintln(os.Stderr, "Use the web terminal at https://ln.bot instead.")
		return fmt.Errorf("not supported in the CLI — use the web terminal")
	},
}

var backupRecoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "Generate a 12-word recovery passphrase",
	Long: `Generate a new recovery passphrase for the active wallet.

The passphrase is shown once — save it somewhere safe. Any previous
recovery passphrase for this wallet becomes invalid.`,
	Example: `  lnbot backup recovery
  lnbot backup recovery --wallet agent02`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		backup, err := ln.Backup.Recovery(context.Background())
		if err != nil {
			return apiError("generating recovery passphrase", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(backup)
		}

		printWarning("Recovery passphrase (save this — shown only once):")
		fmt.Printf("  %s\n", backup.Passphrase)
		fmt.Println()
		fmt.Println("  Any previous recovery passphrase is now invalid.")
		return nil
	},
}
