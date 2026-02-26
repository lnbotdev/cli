package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/format"
)

var payCmd = &cobra.Command{
	Use:   "pay <address-or-bolt11>",
	Short: "Send sats to a Lightning address or BOLT11 invoice",
	Long: `Send sats via the Lightning Network. The target can be:

  - A Lightning address (user@domain) — requires --amount
  - A BOLT11 invoice (starts with lnbc/lntb/lnbs) — amount is encoded

A confirmation prompt is shown before sending. Use --yes to skip it.`,
	Example: `  # Pay a Lightning address
  lnbot pay alice@ln.bot --amount 1000

  # Pay a BOLT11 invoice (amount is in the invoice)
  lnbot pay lnbc10u1pj9x...

  # Skip the confirmation prompt
  lnbot pay alice@ln.bot --amount 500 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		target := args[0]
		params := &lnbot.CreatePaymentParams{Target: target}

		lower := strings.ToLower(target)
		isBolt11 := strings.HasPrefix(lower, "lnbc") ||
			strings.HasPrefix(lower, "lntb") ||
			strings.HasPrefix(lower, "lnbs")
		isAddress := strings.Contains(target, "@")

		amount, _ := cmd.Flags().GetInt64("amount")
		maxFee, _ := cmd.Flags().GetInt64("max-fee")

		if amount > 0 {
			params.Amount = lnbot.Ptr(amount)
		} else if isAddress {
			return fmt.Errorf("--amount is required when paying a Lightning address\n\n  lnbot pay %s --amount <sats>", target)
		}

		if maxFee > 0 {
			params.MaxFee = lnbot.Ptr(maxFee)
		}

		if !isBolt11 && !isAddress {
			return fmt.Errorf("unrecognized target: %s\n\nTarget must be a Lightning address (user@domain) or BOLT11 invoice (lnbc...)", format.Truncate(target, 40))
		}

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		if !yesFlag {
			desc := format.Truncate(target, 50)
			if amount > 0 {
				if !confirm(fmt.Sprintf("Send %s to %s?", format.Sats(amount), desc)) {
					fmt.Println("Cancelled.")
					return nil
				}
			} else {
				if !confirm(fmt.Sprintf("Pay %s?", desc)) {
					fmt.Println("Cancelled.")
					return nil
				}
			}
		}

		start := time.Now()
		payment, err := ln.Payments.Create(context.Background(), params)
		if err != nil {
			return apiError("sending payment", err)
		}
		elapsed := time.Since(start)

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(payment)
		}

		switch payment.Status {
		case "settled":
			if elapsed < 100*time.Millisecond {
				printSuccess("Sent! Settled instantly")
			} else {
				printSuccess(fmt.Sprintf("Sent! Settled in %dms", elapsed.Milliseconds()))
			}
			fmt.Printf("  amount:  %s\n", format.Sats(payment.Amount))
			if payment.ActualFee != nil && *payment.ActualFee > 0 {
				fmt.Printf("  fee:     %s\n", format.Sats(*payment.ActualFee))
			}
			w, err := ln.Wallets.Current(context.Background())
			if err == nil {
				fmt.Printf("  balance: %s\n", format.Sats(w.Available))
			}
		case "failed":
			reason := "unknown"
			if payment.FailureReason != nil {
				reason = *payment.FailureReason
			}
			fmt.Fprintf(os.Stderr, "✗ Payment failed: %s\n", reason)
			fmt.Fprintln(os.Stderr, "  No sats were deducted.")
		default:
			fmt.Printf("  status: %s\n", payment.Status)
		}
		return nil
	},
}

func init() {
	payCmd.Flags().Int64("amount", 0, "amount in sats (required for Lightning addresses)")
	payCmd.Flags().Int64("max-fee", 0, "maximum routing fee in sats")
}
