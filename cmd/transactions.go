package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/format"
)

var transactionsCmd = &cobra.Command{
	Use:     "transactions",
	Short:   "List all transaction history",
	Aliases: []string{"txns", "tx"},
	Long: `Show the combined transaction ledger for the active wallet — both
incoming (credits) and outgoing (debits), newest first.`,
	Example: `  lnbot transactions
  lnbot tx --limit 5
  lnbot transactions --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		txs, err := ln.Transactions.List(context.Background(), &lnbot.ListTransactionsParams{
			Limit: lnbot.Ptr(limit),
		})
		if err != nil {
			return apiError("listing transactions", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(txs)
		}

		if len(txs) == 0 {
			fmt.Println("No transactions yet.")
			return nil
		}

		for _, tx := range txs {
			sign := "+"
			if tx.Type == "debit" {
				sign = "-"
			}
			fmt.Printf("  %-6s  %s%10s sats  bal: %10s  %s\n",
				tx.Type,
				sign,
				format.SatsPlain(tx.Amount),
				format.SatsPlain(tx.BalanceAfter),
				format.TimeAgo(tx.CreatedAt),
			)
		}

		if len(txs) == limit {
			fmt.Printf("\n  %d shown — use --limit to see more\n", limit)
		}
		return nil
	},
}

func init() {
	transactionsCmd.Flags().Int("limit", 20, "max number of results")
}
