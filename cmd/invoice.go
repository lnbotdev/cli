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

var invoiceCmd = &cobra.Command{
	Use:   "invoice <command>",
	Short: "Create and list Lightning invoices",
	Long: `Create invoices to receive bitcoin, and list past invoices.

When you create an invoice the CLI prints its BOLT11 string and waits
for payment via Server-Sent Events. Press Ctrl+C to stop waiting.`,
}

func init() {
	invoiceCreateCmd.Flags().Int64("amount", 0, "Amount to receive in sats, so 1000 is ₿1,000 (required)")
	invoiceCreateCmd.MarkFlagRequired("amount")
	invoiceCreateCmd.Flags().String("memo", "", "Short description attached to the invoice")
	invoiceCreateCmd.Flags().Bool("no-wait", false, "Return immediately without waiting for payment")

	invoiceListCmd.Flags().Int("limit", 20, "Max number of results")
	invoiceListCmd.Flags().Int("after", 0, "Show results after this invoice number (for pagination)")

	invoiceCmd.AddCommand(invoiceCreateCmd)
	invoiceCmd.AddCommand(invoiceListCmd)
}

var invoiceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Lightning invoice to receive bitcoin",
	Long: `Create a new Lightning invoice for the given amount.

Prints the BOLT11 string and waits for the payment to settle via SSE.
Use --no-wait to return immediately. The invoice is valid for 60 minutes.
An invoice paid just before it expires can still settle, so after it
expires the CLI keeps checking for another minute.`,
	Example: `  lnbot invoice create --amount 1000
  lnbot invoice create --amount 5000 --memo "for coffee"
  lnbot invoice create --amount 100 --no-wait
  lnbot invoice create --amount 100 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		amount, _ := cmd.Flags().GetInt64("amount")
		if amount <= 0 {
			return fmt.Errorf("--amount must be a positive integer")
		}

		memo, _ := cmd.Flags().GetString("memo")

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		params := &lnbot.CreateInvoiceParams{Amount: amount}
		if memo != "" {
			params.Memo = lnbot.Ptr(memo)
		}

		ctx := context.Background()
		invoice, err := w.Invoices.Create(ctx, params)
		if err != nil {
			return apiError("creating invoice", err)
		}

		noWait, _ := cmd.Flags().GetBool("no-wait")

		if jsonFlag {
			if noWait {
				return json.NewEncoder(os.Stdout).Encode(invoice)
			}
			final, err := waitForInvoice(ctx, w, invoice.Number, nil)
			if err != nil {
				return json.NewEncoder(os.Stdout).Encode(invoice)
			}
			return json.NewEncoder(os.Stdout).Encode(final)
		}

		fmt.Printf("  Amount:  %s\n", format.BTC(invoice.Amount))
		fmt.Printf("  Status:  %s\n", format.Cap(invoice.Status))
		fmt.Println("  BOLT11:")
		fmt.Printf("  %s\n", invoice.Bolt11)
		fmt.Println()

		if noWait {
			return nil
		}

		fmt.Print("  Waiting for payment... (Ctrl+C to stop)")

		final, err := waitForInvoice(ctx, w, invoice.Number, func() {
			fmt.Println()
			fmt.Print("  Invoice expired. Checking for a payment made just before it did...")
		})
		fmt.Println()
		if err != nil {
			return err
		}
		if final.Status == "settled" {
			printSuccess(fmt.Sprintf("Payment received: +%s", format.BTC(final.Amount)))
			return nil
		}
		fmt.Println("  No payment arrived.")
		return nil
	},
}

var invoiceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List invoices",
	Aliases: []string{"ls"},
	Long:    `Show recent invoices for the active wallet, newest first.`,
	Example: `  lnbot invoice list
  lnbot invoice list --limit 5
  lnbot invoice list --after 20
  lnbot invoice list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		limit, _ := cmd.Flags().GetInt("limit")
		after, _ := cmd.Flags().GetInt("after")

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		params := &lnbot.ListInvoicesParams{Limit: lnbot.Ptr(limit)}
		if after > 0 {
			params.After = lnbot.Ptr(after)
		}

		invoices, err := w.Invoices.List(context.Background(), params)
		if err != nil {
			return apiError("listing invoices", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(invoices)
		}

		if len(invoices) == 0 {
			fmt.Println("No invoices yet.")
			return nil
		}

		for _, inv := range invoices {
			fmt.Printf("  #%4d  %-8s  %12s  %s\n",
				inv.Number,
				format.Cap(inv.Status),
				format.BTC(inv.Amount),
				format.TimeAgo(inv.CreatedAt),
			)
		}

		if len(invoices) == limit {
			last := invoices[len(invoices)-1].Number
			fmt.Printf("\n  %d shown. Next page: --after %d\n", limit, last)
		}
		return nil
	},
}
