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
  lnbot status --wallet agent02
  lnbot status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		ln, _, name, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		ctx := context.Background()

		t0 := time.Now()
		w, err := ln.Wallets.Current(ctx)
		if err != nil {
			return apiError("fetching status", err)
		}
		latency := time.Since(t0)

		var firstAddr string
		addrs, addrErr := ln.Addresses.List(ctx)
		if addrErr == nil && len(addrs) > 0 {
			firstAddr = addrs[0].Address
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]any{
				"wallet":    name,
				"walletId":  w.WalletID,
				"balance":   w.Balance,
				"available": w.Available,
				"onHold":    w.OnHold,
				"address":   firstAddr,
				"latencyMs": latency.Milliseconds(),
			})
		}

		fmt.Printf("  name:      %s\n", name)
		fmt.Printf("  id:        %s\n", w.WalletID)
		if firstAddr != "" {
			fmt.Printf("  address:   %s\n", firstAddr)
		}
		fmt.Printf("  balance:   %s\n", format.Sats(w.Balance))
		fmt.Printf("  available: %s\n", format.Sats(w.Available))
		fmt.Printf("  on hold:   %s\n", format.Sats(w.OnHold))
		fmt.Printf("  api:       ✓ connected (%dms)\n", latency.Milliseconds())
		return nil
	},
}
