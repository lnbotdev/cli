package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/lnbotdev/cli/internal/format"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Wallet status and API health",
	Long:  `Show wallet details, balance, addresses, and API connectivity.`,
	Example: `  lnbot status
  lnbot status --wallet wal_abc
  lnbot status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := resolveWallet()
		if err != nil {
			return err
		}

		ctx := context.Background()

		t0 := time.Now()
		wal, err := w.Get(ctx)
		if err != nil {
			return apiError("fetching status", err)
		}
		latency := time.Since(t0)

		var firstAddr string
		addrs, addrErr := w.Addresses.List(ctx)
		if addrErr == nil && len(addrs) > 0 {
			firstAddr = addrs[0].Address
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{
				"walletId":  wal.WalletID,
				"name":      wal.Name,
				"balance":   wal.Balance,
				"available": wal.Available,
				"onHold":    wal.OnHold,
				"address":   firstAddr,
				"latencyMs": latency.Milliseconds(),
			})
		}

		fmt.Printf("  name:      %s\n", wal.Name)
		fmt.Printf("  id:        %s\n", wal.WalletID)
		if firstAddr != "" {
			fmt.Printf("  address:   %s\n", firstAddr)
		}
		fmt.Printf("  balance:   %s\n", format.Sats(wal.Balance))
		fmt.Printf("  available: %s\n", format.Sats(wal.Available))
		fmt.Printf("  on hold:   %s\n", format.Sats(wal.OnHold))
		fmt.Printf("  api:       ✓ connected (%dms)\n", latency.Milliseconds())
		return nil
	},
}
