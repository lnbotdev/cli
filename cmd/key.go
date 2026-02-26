package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
)

var keyCmd = &cobra.Command{
	Use:   "key <command>",
	Short: "Show or rotate API keys",
	Long: `View and rotate the API keys for your wallet.

Each wallet has two key slots: primary (0) and secondary (1).
Rotating a key revokes the old one immediately.`,
}

func init() {
	keyCmd.AddCommand(keyShowCmd)
	keyCmd.AddCommand(keyRotateCmd)
}

var keyShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show API keys from local config",
	Long:  `Print the primary and secondary API keys stored in the local config.`,
	Example: `  lnbot key show
  lnbot key show --wallet agent02
  lnbot key show --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		entry, _, err := cfg.ResolveWallet(walletFlag)
		if err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"primary_key":   entry.PrimaryKey,
				"secondary_key": entry.SecondaryKey,
			})
		}

		fmt.Printf("  primary:   %s\n", entry.PrimaryKey)
		if entry.SecondaryKey != "" {
			fmt.Printf("  secondary: %s\n", entry.SecondaryKey)
		}
		return nil
	},
}

var keyRotateCmd = &cobra.Command{
	Use:   "rotate <slot>",
	Short: "Rotate an API key",
	Long: `Rotate the API key at the given slot. The old key is revoked immediately.

Slots:
  0  primary key
  1  secondary key

The new key is printed once — save it. The local config is updated automatically.`,
	Example: `  lnbot key rotate 0
  lnbot key rotate 1 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		slot, err := strconv.Atoi(args[0])
		if err != nil || (slot != 0 && slot != 1) {
			return fmt.Errorf("slot must be 0 (primary) or 1 (secondary)")
		}

		slotLabel := "primary"
		if slot == 1 {
			slotLabel = "secondary"
		}

		if !yesFlag {
			if !confirm(fmt.Sprintf("Rotate %s key? The old key will stop working.", slotLabel)) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		ln, entry, name, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		sdkSlot := slot + 1

		rotated, err := ln.Keys.Rotate(context.Background(), sdkSlot)
		if err != nil {
			return apiError("rotating key", err)
		}

		if slot == 0 {
			entry.PrimaryKey = rotated.Key
		} else {
			entry.SecondaryKey = rotated.Key
		}

		cfg.Wallets[name] = *entry
		if err := cfg.Save(); err != nil {
			return err
		}

		printSuccess(fmt.Sprintf("%s key rotated", slotLabel))
		fmt.Printf("  key: %s\n", rotated.Key)
		fmt.Println()
		fmt.Println("  Save this — it won't be shown again.")
		return nil
	},
}
