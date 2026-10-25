package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

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
	paymentListCmd.Flags().Int("limit", 20, "Max number of results")
	paymentListCmd.Flags().Int("after", 0, "Show results after this payment number (for pagination)")

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
		limit, _ := cmd.Flags().GetInt("limit")
		after, _ := cmd.Flags().GetInt("after")

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		params := &lnbot.ListPaymentsParams{Limit: lnbot.Ptr(limit)}
		if after > 0 {
			params.After = lnbot.Ptr(after)
		}

		payments, err := w.Payments.List(context.Background(), params)
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

		labels := make([]string, len(payments))
		width := 10
		for i := range payments {
			labels[i] = paymentState(&payments[i])
			width = max(width, len([]rune(labels[i])))
		}

		for i, p := range payments {
			addr := p.Address
			if addr == "" {
				addr = "--"
			}
			fmt.Printf("  #%4d  %-*s  %12s  %8s  %s\n",
				p.Number,
				width, labels[i],
				format.BTC(p.Amount),
				format.TimeAgo(p.CreatedAt),
				format.Truncate(addr, 40),
			)
		}

		if len(payments) == limit {
			last := payments[len(payments)-1].Number
			fmt.Printf("\n  %d shown. Next page: --after %d\n", limit, last)
		}
		return nil
	},
}

// paymentState is a payment's status, with its stage while it is in progress:
// "Pending · retrying", "Processing · in flight".
func paymentState(p *lnbot.Payment) string {
	state := format.Cap(p.Status)
	if p.Stage != "" && !final(p) && p.Stage != p.Status {
		state += " · " + strings.ReplaceAll(p.Stage, "_", " ")
	}
	return state
}
