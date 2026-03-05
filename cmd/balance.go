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
  lnbot balance --wallet wal_abc
  lnbot balance --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := resolveWallet()
		if err != nil {
			return err
		}

		wal, err := w.Get(context.Background())
		if err != nil {
			return apiError("fetching balance", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(wal)
		}

		fmt.Printf("  balance:   %s\n", format.Sats(wal.Balance))
		fmt.Printf("  available: %s\n", format.Sats(wal.Available))
		fmt.Printf("  on hold:   %s\n", format.Sats(wal.OnHold))
		return nil
	},
}
