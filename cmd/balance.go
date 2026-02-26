package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lnbotdev/cli/internal/format"
)

var balanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Show wallet balance",
	Long:  `Display the current balance, available amount, and on-hold amount for the active wallet.`,
	Example: `  lnbot balance
  lnbot balance --wallet agent02
  lnbot balance --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		w, err := ln.Wallets.Current(context.Background())
		if err != nil {
			return apiError("fetching balance", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(w)
		}

		fmt.Printf("  balance:   %s\n", format.Sats(w.Balance))
		fmt.Printf("  available: %s\n", format.Sats(w.Available))
		fmt.Printf("  on hold:   %s\n", format.Sats(w.OnHold))
		return nil
	},
}
