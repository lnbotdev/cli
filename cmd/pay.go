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
	Short: "Send bitcoin to a Lightning address, LNURL or BOLT11 invoice",
	Long: `Send bitcoin over the Lightning Network. The target can be:

  - A Lightning address (user@domain): needs --amount or --all
  - An LNURL (lnurl1...): needs --amount or --all
  - A BOLT11 invoice (lnbc..., lntb..., lnbs...): the amount is in the invoice

Before it asks, pay shows the amount, the fees and the total. Payments
between ln.bot wallets are free. A payment out to Lightning pays a service
fee of 0.25% (at least ₿1) and the network fee, which is the Lightning
routing fee. The network fee is capped at 1% of the amount, at least ₿20,
unless you set --max-fee.

Use --yes to skip the fee quote and the confirmation prompt. The CLI waits
for settlement; use --no-wait to return immediately.`,
	Example: `  # Pay a Lightning address
  lnbot pay alice@ln.bot --amount 1000

  # Pay a BOLT11 invoice (the amount is in the invoice)
  lnbot pay lnbc10u1pj9x...

  # Pay an LNURL
  lnbot pay lnurl1dp68gurn8ghj7... --amount 500

  # Send everything the wallet can to a Lightning address
  lnbot pay alice@example.com --all

  # Allow a higher network fee than the default cap
  lnbot pay alice@example.com --amount 2000 --max-fee 100

  # Return immediately without waiting for settlement
  lnbot pay alice@ln.bot --amount 500 --no-wait

  # Skip the fee quote and the confirmation prompt
  lnbot pay alice@ln.bot --amount 500 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		all, _ := cmd.Flags().GetBool("all")

		if all {
			switch {
			case amount > 0:
				return fmt.Errorf("use --amount or --all, not both")
			case maxFee > 0:
				return fmt.Errorf("--all sends what the default network fee cap allows; leave out --max-fee")
			case isBolt11:
				return fmt.Errorf("--all works with a Lightning address or LNURL; a BOLT11 invoice sets its own amount")
			}
		}

		if amount > 0 {
			params.Amount = lnbot.Ptr(amount)
		} else if (isAddress || isLNURL) && !all {
			return fmt.Errorf("--amount is required when paying a Lightning address or LNURL (or --all to send everything)\n\n  lnbot pay %s --amount <sats>", format.Truncate(target, 40))
		}

		if maxFee > 0 {
			params.MaxFee = lnbot.Ptr(maxFee)
		}

		if !isBolt11 && !isAddress && !isLNURL {
			return fmt.Errorf("unrecognized target: %s\n\nTarget must be a Lightning address (user@domain), LNURL (lnurl1...), or BOLT11 invoice (lnbc...)", format.Truncate(target, 40))
		}

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		ctx := context.Background()

		if all {
			res, err := w.Payments.Resolve(ctx, target)
			if err != nil {
				return apiError("checking recipient", err)
			}
			if res.MaxSendable == nil || *res.MaxSendable <= 0 {
				msg := "the balance doesn't cover this recipient's minimum after fees"
				if res.Min != nil {
					msg += fmt.Sprintf(" (%s)", format.BTC(*res.Min))
				}
				return fmt.Errorf("%s", msg)
			}
			amount = *res.MaxSendable
			params.Amount = lnbot.Ptr(amount)
		}

		if !yesFlag {
			quote, err := quotePayment(ctx, w, target, amount)
			if err != nil {
				return apiError("pricing payment", err)
			}
			printQuote(quote)
			if quote.FeeWarning != nil && maxFee == 0 {
				return fmt.Errorf("the network fee is above the default cap; pass --max-fee <sats> to allow it")
			}
			if quote.Amount != nil {
				amount = *quote.Amount
			}

			desc := format.Truncate(target, 50)
			prompt := fmt.Sprintf("Pay %s?", desc)
			if amount > 0 {
				prompt = fmt.Sprintf("Send %s to %s?", format.BTC(amount), desc)
			}
			if !confirm(prompt) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		start := time.Now()
		payment, err := w.Payments.Create(ctx, params)
		if err != nil {
			return apiError("sending payment", err)
		}

		noWait, _ := cmd.Flags().GetBool("no-wait")

		if jsonFlag {
			if !noWait && (payment.Status == "pending" || payment.Status == "processing") {
				payment, err = waitForPaymentJSON(ctx, w, payment)
				if err != nil {
					return json.NewEncoder(os.Stdout).Encode(payment)
				}
			}
			return json.NewEncoder(os.Stdout).Encode(payment)
		}

		if noWait {
			fmt.Printf("  Status:  %s\n", format.Cap(payment.Status))
			fmt.Printf("  Number:  %d\n", payment.Number)
			return nil
		}

		return printPaymentResult(ctx, w, payment, start)
	},
}

// quotePayment prices the payment: a BOLT11 invoice at its own amount, a
// Lightning address or LNURL at amount.
func quotePayment(ctx context.Context, w *lnbot.WalletHandle, target string, amount int64) (*lnbot.ResolveTargetResponse, error) {
	if amount > 0 {
		return w.Payments.ResolveWithAmount(ctx, target, amount)
	}
	return w.Payments.Resolve(ctx, target)
}

// printQuote shows what the payment will cost before it is sent.
func printQuote(q *lnbot.ResolveTargetResponse) {
	if q.Amount == nil {
		return
	}
	rows := [][2]string{{"Amount:", format.BTC(*q.Amount)}}
	estimated := q.NetworkFeeEstimate != nil
	if q.Internal {
		rows = append(rows, [2]string{"Fees:", "₿0 (to an ln.bot wallet)"})
	} else {
		switch {
		case estimated:
			rows = append(rows, [2]string{"Network fee:", format.BTC(*q.NetworkFeeEstimate) + " (estimate)"})
		case q.MaxFee != nil:
			rows = append(rows, [2]string{"Network fee:", "Up to " + format.BTC(*q.MaxFee)})
		}
		if q.ServiceFee != nil {
			rows = append(rows, [2]string{"Service fee:", format.BTC(*q.ServiceFee)})
		}
	}
	if q.Total != nil {
		total := format.BTC(*q.Total)
		if !q.Internal && !estimated {
			total = "Up to " + total
		}
		rows = append(rows, [2]string{"Total:", total})
	}

	width := 0
	for _, r := range rows {
		width = max(width, len(r[0]))
	}
	for _, r := range rows {
		fmt.Printf("  %-*s  %s\n", width, r[0], r[1])
	}
	if q.FeeWarning != nil {
		printWarning(*q.FeeWarning)
	}
}

// paymentFees is what a settled payment cost on top of its amount: the
// network fee charged plus the service fee.
func paymentFees(p *lnbot.Payment) (network, total int64) {
	if p.ActualFee != nil {
		network = *p.ActualFee
	}
	return network, network + p.ServiceFee
}

func feeSummary(p *lnbot.Payment) string {
	network, total := paymentFees(p)
	if total == 0 {
		return format.BTC(0)
	}
	return fmt.Sprintf("%s (network %s, service %s)", format.BTC(total), format.BTC(network), format.BTC(p.ServiceFee))
}

func printPaymentResult(ctx context.Context, w *lnbot.WalletHandle, payment *lnbot.Payment, start time.Time) error {
	switch payment.Status {
	case "settled":
		elapsed := time.Since(start)
		if elapsed < 100*time.Millisecond {
			printSuccess("Payment settled instantly")
		} else {
			printSuccess(fmt.Sprintf("Payment settled in %dms", elapsed.Milliseconds()))
		}
		fmt.Printf("  Amount:     %s\n", format.BTC(payment.Amount))
		fmt.Printf("  Fees:       %s\n", feeSummary(payment))
		// What left the balance, as the transaction ledger shows it.
		if _, fees := paymentFees(payment); fees > 0 {
			fmt.Printf("  Total:      %s\n", format.BTC(payment.Amount+fees))
		}
		wal, err := w.Get(ctx)
		if err == nil {
			fmt.Printf("  Available:  %s\n", format.BTC(wal.Available))
		}
	case "failed":
		reason := "unknown"
		if payment.FailureReason != nil {
			reason = *payment.FailureReason
		}
		fmt.Fprintf(os.Stderr, "✗ Payment failed: %s\n", reason)
		fmt.Fprintln(os.Stderr, "  Nothing was deducted.")
	default:
		// An API that reports stages gets a line per stage; one that doesn't,
		// the single waiting line it always had.
		staged := payment.Stage != ""
		if staged {
			printStage(payment)
			fmt.Println("  Waiting for settlement... (Ctrl+C to stop)")
		} else {
			fmt.Print("  Waiting for settlement... (Ctrl+C to stop)")
		}
		result, err := followPayment(ctx, w, payment, printStage)
		if !staged {
			fmt.Println()
		}
		if err != nil {
			return err
		}
		if final(result) {
			return printPaymentResult(ctx, w, result, start)
		}
		// The wait ran out first. A pending payment is retried for minutes, a
		// processing one waits for the Lightning network: say only where it is.
		fmt.Println("  Still in progress. Check it with 'lnbot payment list'.")
		if result.StatusDetail != nil && *result.StatusDetail != "" {
			fmt.Printf("  %s\n", *result.StatusDetail)
		}
	}
	return nil
}

// printStage shows where a payment in progress is, in the API's words, with
// the time it is retried until in local time.
func printStage(p *lnbot.Payment) {
	line := "  " + format.Stage(p.Stage)
	if p.StatusDetail != nil && *p.StatusDetail != "" {
		line += ": " + *p.StatusDetail
	}
	if p.RetryUntil != nil {
		line += " (until " + clock(*p.RetryUntil) + ")"
	}
	fmt.Println(line)
}

func waitForPaymentJSON(ctx context.Context, w *lnbot.WalletHandle, payment *lnbot.Payment) (*lnbot.Payment, error) {
	result, err := waitPayment(ctx, w, payment.Number)
	if err != nil {
		return payment, err
	}
	return result, nil
}

func init() {
	payCmd.Flags().Int64("amount", 0, "Amount to send in sats, so 1000 is ₿1,000 (required for Lightning addresses and LNURLs)")
	payCmd.Flags().Int64("max-fee", 0, "Most to pay in network fees, in sats (default: 1% of the amount, at least 20)")
	payCmd.Flags().Bool("all", false, "Send everything the wallet can, after fees (Lightning addresses and LNURLs)")
	payCmd.Flags().Bool("no-wait", false, "Return as soon as the API accepts the payment")
}
