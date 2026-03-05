package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current wallet info",
	Long:  `Print the active wallet's ID, name, Lightning address, and truncated API key.`,
	Example: `  lnbot whoami
  lnbot whoami --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := resolveWallet()
		if err != nil {
			return err
		}

		ctx := context.Background()

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"wallet_id": w.WalletID,
				"api_key":   truncateKey(cfg.PrimaryKey),
			})
		}

		fmt.Printf("  wallet:  %s\n", w.WalletID)

		wal, err := w.Get(ctx)
		if err == nil {
			fmt.Printf("  name:    %s\n", wal.Name)
		}

		addrs, err := w.Addresses.List(ctx)
		if err == nil && len(addrs) > 0 {
			fmt.Printf("  address: %s\n", addrs[0].Address)
		}

		fmt.Printf("  api_key: %s\n", truncateKey(cfg.PrimaryKey))
		return nil
	},
}
