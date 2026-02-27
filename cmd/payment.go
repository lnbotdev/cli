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

var paymentCmd = &cobra.Command{
	Use:   "payment <command>",
	Short: "List outgoing payments",
	Long:  `View outgoing payments sent from the active wallet.`,
}

func init() {
	paymentListCmd.Flags().Int("limit", 20, "max number of results")
	paymentListCmd.Flags().Int("after", 0, "show results after this payment number (for pagination)")

	paymentCmd.AddCommand(paymentListCmd)
}

var paymentListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List outgoing payments",
	Aliases: []string{"ls"},
	Long:    `Show recent outgoing payments for the active wallet, newest first.`,
	Example: `  lnbot payment list
  lnbot payment list --limit 5
  lnbot payment list --after 20
  lnbot payment list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")
		after, _ := cmd.Flags().GetInt("after")

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		params := &lnbot.ListPaymentsParams{Limit: lnbot.Ptr(limit)}
		if after > 0 {
			params.After = lnbot.Ptr(after)
		}

		payments, err := ln.Payments.List(context.Background(), params)
		if err != nil {
			return apiError("listing payments", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(payments)
		}

		if len(payments) == 0 {
			fmt.Println("No payments yet.")
			return nil
		}

		for _, p := range payments {
			addr := p.Address
			if addr == "" {
				addr = "--"
			}
			fmt.Printf("  #%4d  %-8s  %10s sats  %8s  %s\n",
				p.Number,
				p.Status,
				format.SatsPlain(p.Amount),
				format.TimeAgo(p.CreatedAt),
				format.Truncate(addr, 40),
			)
		}

		if len(payments) == limit {
			last := payments[len(payments)-1].Number
			fmt.Printf("\n  %d shown — next page: --after %d\n", limit, last)
		}
		return nil
	},
}
