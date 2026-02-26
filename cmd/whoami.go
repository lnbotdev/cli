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
		if err := requireConfig(); err != nil {
			return err
		}

		ln, entry, name, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(map[string]string{
				"wallet":  entry.ID,
				"name":    name,
				"api_key": truncateKey(entry.PrimaryKey),
			})
		}

		fmt.Printf("  wallet:  %s\n", entry.ID)
		fmt.Printf("  name:    %s\n", name)

		addrs, err := ln.Addresses.List(context.Background())
		if err == nil && len(addrs) > 0 {
			fmt.Printf("  address: %s\n", addrs[0].Address)
		}

		fmt.Printf("  api_key: %s\n", truncateKey(entry.PrimaryKey))
		return nil
	},
}
