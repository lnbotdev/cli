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
	Use:   "pay <target>",
	Short: "Send sats to a Lightning address, LNURL, or BOLT11 invoice",
	Long: `Send sats via the Lightning Network. The target can be:

  - A Lightning address (user@domain) — requires --amount
  - An LNURL (lnurl1...) — requires --amount
  - A BOLT11 invoice (starts with lnbc/lntb/lnbs) — amount is encoded

A confirmation prompt is shown before sending. Use --yes to skip it.
The CLI waits for settlement via SSE. Use --no-wait to return immediately.`,
	Example: `  # Pay a Lightning address
  lnbot pay alice@ln.bot --amount 1000

  # Pay a BOLT11 invoice (amount is in the invoice)
  lnbot pay lnbc10u1pj9x...

  # Pay an LNURL
  lnbot pay lnurl1dp68gurn8ghj7... --amount 500

  # Return immediately without waiting for settlement
  lnbot pay alice@ln.bot --amount 500 --no-wait

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
		isLNURL := strings.HasPrefix(lower, "lnurl")

		amount, _ := cmd.Flags().GetInt64("amount")
		maxFee, _ := cmd.Flags().GetInt64("max-fee")

		if amount > 0 {
			params.Amount = lnbot.Ptr(amount)
		} else if isAddress || isLNURL {
			return fmt.Errorf("--amount is required when paying a Lightning address or LNURL\n\n  lnbot pay %s --amount <sats>", format.Truncate(target, 40))
		}

		if maxFee > 0 {
			params.MaxFee = lnbot.Ptr(maxFee)
		}

		if !isBolt11 && !isAddress && !isLNURL {
			return fmt.Errorf("unrecognized target: %s\n\nTarget must be a Lightning address (user@domain), LNURL (lnurl1...), or BOLT11 invoice (lnbc...)", format.Truncate(target, 40))
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

		ctx := context.Background()
		start := time.Now()
		payment, err := ln.Payments.Create(ctx, params)
		if err != nil {
			return apiError("sending payment", err)
		}

		noWait, _ := cmd.Flags().GetBool("no-wait")

		if jsonFlag {
			if !noWait && (payment.Status == "pending" || payment.Status == "processing") {
				payment, err = waitForPaymentJSON(ctx, ln, payment)
				if err != nil {
					return json.NewEncoder(os.Stdout).Encode(payment)
				}
			}
			return json.NewEncoder(os.Stdout).Encode(payment)
		}

		if noWait {
			fmt.Printf("  status: %s\n", payment.Status)
			fmt.Printf("  number: %d\n", payment.Number)
			return nil
		}

		return printPaymentResult(ctx, ln, payment, start)
	},
}

func printPaymentResult(ctx context.Context, ln *lnbot.Client, payment *lnbot.Payment, start time.Time) error {
	switch payment.Status {
	case "settled":
		elapsed := time.Since(start)
		if elapsed < 100*time.Millisecond {
			printSuccess("Sent! Settled instantly")
		} else {
			printSuccess(fmt.Sprintf("Sent! Settled in %dms", elapsed.Milliseconds()))
		}
		fmt.Printf("  amount:  %s\n", format.Sats(payment.Amount))
		if payment.ActualFee != nil && *payment.ActualFee > 0 {
			fmt.Printf("  fee:     %s\n", format.Sats(*payment.ActualFee))
		}
		w, err := ln.Wallets.Current(ctx)
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
		fmt.Print("  Waiting for settlement... (Ctrl+C to stop)")

		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		events, errs := ln.Payments.Watch(watchCtx, payment.Number, nil)
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					fmt.Println()
					return nil
				}
				fmt.Println()
				return printPaymentResult(ctx, ln, &ev.Data, start)
			case err, ok := <-errs:
				if ok && err != nil {
					fmt.Println()
					return err
				}
				return nil
			}
		}
	}
	return nil
}

func waitForPaymentJSON(ctx context.Context, ln *lnbot.Client, payment *lnbot.Payment) (*lnbot.Payment, error) {
	watchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	events, errs := ln.Payments.Watch(watchCtx, payment.Number, nil)
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				return payment, nil
			}
			return &ev.Data, nil
		case err, ok := <-errs:
			if ok && err != nil {
				return payment, err
			}
			return payment, nil
		}
	}
}

func init() {
	payCmd.Flags().Int64("amount", 0, "amount in sats (required for Lightning addresses and LNURLs)")
	payCmd.Flags().Int64("max-fee", 0, "maximum routing fee in sats")
	payCmd.Flags().Bool("no-wait", false, "return immediately without waiting for settlement")
}
