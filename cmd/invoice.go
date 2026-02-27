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
	Long: `Create invoices to receive sats, and list past invoices.

When you create an invoice the CLI prints a QR code and waits for
payment via Server-Sent Events. Press Ctrl+C to stop waiting.`,
}

func init() {
	invoiceCreateCmd.Flags().Int64("amount", 0, "amount in sats (required)")
	invoiceCreateCmd.MarkFlagRequired("amount")
	invoiceCreateCmd.Flags().String("memo", "", "short description attached to the invoice")

	invoiceListCmd.Flags().Int("limit", 20, "max number of results")

	invoiceCmd.AddCommand(invoiceCreateCmd)
	invoiceCmd.AddCommand(invoiceListCmd)
}

var invoiceCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a Lightning invoice to receive sats",
	Long: `Create a new Lightning invoice for the given amount.

Prints the BOLT11 string, renders a QR code in the terminal, and
automatically waits for the payment to settle via SSE. Press Ctrl+C
to stop waiting — the invoice remains valid until it expires.`,
	Example: `  lnbot invoice create --amount 1000
  lnbot invoice create --amount 5000 --memo "for coffee"
  lnbot invoice create --amount 100 --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		amount, _ := cmd.Flags().GetInt64("amount")
		if amount <= 0 {
			return fmt.Errorf("--amount must be a positive integer")
		}

		memo, _ := cmd.Flags().GetString("memo")

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		params := &lnbot.CreateInvoiceParams{Amount: amount}
		if memo != "" {
			params.Memo = lnbot.Ptr(memo)
		}

		ctx := context.Background()
		invoice, err := ln.Invoices.Create(ctx, params)
		if err != nil {
			return apiError("creating invoice", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(invoice)
		}

		fmt.Printf("  amount:  %s\n", format.Sats(invoice.Amount))
		fmt.Printf("  status:  %s\n", invoice.Status)
		fmt.Println("  bolt11:")
		fmt.Printf("  %s\n", invoice.Bolt11)
		fmt.Println()

		fmt.Print("  Waiting for payment... (Ctrl+C to stop)")

		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		events, errs := ln.Invoices.Watch(watchCtx, invoice.Number, nil)
		for {
			select {
			case ev, ok := <-events:
				if !ok {
					fmt.Println()
					return nil
				}
				switch ev.Event {
				case "settled":
					fmt.Println()
					printSuccess(fmt.Sprintf("Payment received! +%s", format.Sats(invoice.Amount)))
					return nil
				case "expired":
					fmt.Println()
					fmt.Println("  Invoice expired.")
					return nil
				}
			case err, ok := <-errs:
				if ok && err != nil {
					fmt.Println()
					return err
				}
				return nil
			}
		}
	},
}

var invoiceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List invoices",
	Aliases: []string{"ls"},
	Long:    `Show recent invoices for the active wallet, newest first.`,
	Example: `  lnbot invoice list
  lnbot invoice list --limit 5
  lnbot invoice list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		limit, _ := cmd.Flags().GetInt("limit")

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		invoices, err := ln.Invoices.List(context.Background(), &lnbot.ListInvoicesParams{
			Limit: lnbot.Ptr(limit),
		})
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
			fmt.Printf("  #%4d  %-8s  %10s sats  %s\n",
				inv.Number,
				inv.Status,
				format.SatsPlain(inv.Amount),
				format.TimeAgo(inv.CreatedAt),
			)
		}

		if len(invoices) == limit {
			fmt.Printf("\n  %d shown — use --limit to see more\n", limit)
		}
		return nil
	},
}
