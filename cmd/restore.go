package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
)

var restoreCmd = &cobra.Command{
	Use:   "restore <command>",
	Short: "Restore a wallet from backup",
	Long: `Restore access to a wallet using a backup method.

Two methods are available:
  recovery  — restore using the 12-word passphrase
  passkey   — restore using a registered WebAuthn passkey (browser only)

Restoring rotates all API keys. Old keys stop working immediately.`,
}

func init() {
	restoreRecoveryCmd.Flags().String("passphrase", "", "12-word recovery passphrase")
	restoreRecoveryCmd.MarkFlagRequired("passphrase")

	restoreCmd.AddCommand(restorePasskeyCmd)
	restoreCmd.AddCommand(restoreRecoveryCmd)
}

var restorePasskeyCmd = &cobra.Command{
	Use:   "passkey",
	Short: "Restore via passkey (browser only)",
	Long: `Restore a wallet using a registered WebAuthn passkey.

This requires a browser with WebAuthn support and is not available in
the CLI. Use the web terminal at https://ln.bot instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(os.Stderr, "Passkey authentication requires a browser with WebAuthn support.")
		fmt.Fprintln(os.Stderr, "Use the web terminal at https://ln.bot instead.")
		return fmt.Errorf("not supported in the CLI — use the web terminal")
	},
}

var restoreRecoveryCmd = &cobra.Command{
	Use:   "recovery",
	Short: "Restore a wallet via recovery passphrase",
	Long: `Restore wallet access using the 12-word recovery passphrase.

This rotates all API keys — old keys stop working immediately. If the
wallet already exists in local config it is updated in place.`,
	Example: `  lnbot restore recovery --passphrase "word1 word2 word3 ... word12"`,
	RunE: func(cmd *cobra.Command, args []string) error {
		phrase, _ := cmd.Flags().GetString("passphrase")

		ln := config.AnonClient()
		restored, err := ln.Restore.Recovery(context.Background(), &lnbot.RecoveryRestoreParams{
			Passphrase: phrase,
		})
		if err != nil {
			return apiError("restoring wallet", err)
		}

		if cfg == nil {
			cfg, err = config.Init()
			if err != nil {
				return err
			}
		}

		name := ""
		for n, entry := range cfg.Wallets {
			if entry.ID == restored.WalletID {
				name = n
				break
			}
		}

		if name == "" {
			name = restored.Name
			if name == "" {
				name = "restored"
			}
			if _, exists := cfg.Wallets[name]; exists {
				n := 1
				for {
					candidate := fmt.Sprintf("%s-%d", name, n)
					if _, exists := cfg.Wallets[candidate]; !exists {
						name = candidate
						break
					}
					n++
				}
			}
		}

		cfg.Wallets[name] = config.WalletEntry{
			ID:           restored.WalletID,
			PrimaryKey:   restored.PrimaryKey,
			SecondaryKey: restored.SecondaryKey,
		}
		cfg.Active = name
		if err := cfg.Save(); err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(restored)
		}

		printSuccess("Wallet restored")
		fmt.Printf("  id:   %s\n", restored.WalletID)
		fmt.Printf("  name: %s\n", name)

		newClient := lnbot.New(restored.PrimaryKey)
		addrs, err := newClient.Addresses.List(context.Background())
		if err == nil && len(addrs) > 0 {
			fmt.Printf("  address: %s\n", addrs[0].Address)
		}

		return nil
	},
}
