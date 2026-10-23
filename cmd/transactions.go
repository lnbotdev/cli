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
	Long: `Show the combined transaction ledger for the active wallet: both
incoming (credits) and outgoing (debits), newest first.`,
	Example: `  lnbot transactions
  lnbot tx --limit 5
  lnbot transactions --after 20
  lnbot transactions --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		after, _ := cmd.Flags().GetInt("after")

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		params := &lnbot.ListTransactionsParams{Limit: lnbot.Ptr(limit)}
		if after > 0 {
			params.After = lnbot.Ptr(after)
		}

		txs, err := w.Transactions.List(context.Background(), params)
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
			fmt.Printf("  %-6s  %12s  Balance %12s  %s\n",
				format.Cap(tx.Type),
				sign+format.BTC(tx.Amount),
				format.BTC(tx.BalanceAfter),
				format.TimeAgo(tx.CreatedAt),
			)
		}

		if len(txs) == limit {
			last := txs[len(txs)-1].Number
			fmt.Printf("\n  %d shown. Next page: --after %d\n", limit, last)
		}
		return nil
	},
}

func init() {
	transactionsCmd.Flags().Int("limit", 20, "Max number of results")
	transactionsCmd.Flags().Int("after", 0, "Show results after this transaction number (for pagination)")
}
