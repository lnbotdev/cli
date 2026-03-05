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

var addressCmd = &cobra.Command{
	Use:   "address <command>",
	Short: "Manage Lightning addresses",
	Long: `Buy, list, transfer, and delete Lightning addresses.

Every wallet gets a free auto-generated address (e.g. x8km2n@ln.bot).
You can also buy vanity addresses like alice@ln.bot.`,
}

func init() {
	addressTransferCmd.Flags().String("target-key", "", "target wallet API key")

	addressCmd.AddCommand(addressListCmd)
	addressCmd.AddCommand(addressBuyCmd)
	addressCmd.AddCommand(addressTransferCmd)
	addressCmd.AddCommand(addressDeleteCmd)
}

var addressListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List Lightning addresses",
	Aliases: []string{"ls"},
	Long:    `Show all Lightning addresses for the active wallet.`,
	Example: `  lnbot address list
  lnbot address list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := resolveWallet()
		if err != nil {
			return err
		}

		addrs, err := w.Addresses.List(context.Background())
		if err != nil {
			return apiError("listing addresses", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(addrs)
		}

		if len(addrs) == 0 {
			fmt.Println("No addresses yet.")
			return nil
		}

		for _, a := range addrs {
			tag := "vanity"
			if a.Generated {
				tag = "generated"
			} else if a.Cost > 0 {
				tag = fmt.Sprintf("vanity, %s", format.Sats(a.Cost))
			}
			fmt.Printf("  %s  (%s)\n", a.Address, tag)
		}
		return nil
	},
}

var addressBuyCmd = &cobra.Command{
	Use:   "buy <name>",
	Short: "Buy a vanity Lightning address",
	Long: `Claim a vanity Lightning address like alice@ln.bot.

The cost (if any) is deducted from the wallet balance. Plus-addressing
is included automatically (alice+anything@ln.bot).`,
	Example: `  lnbot address buy alice
  lnbot address buy mybot --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		if !yesFlag {
			if !confirm(fmt.Sprintf("Claim %s@ln.bot?", name)) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		addr, err := w.Addresses.Create(context.Background(), &lnbot.CreateAddressParams{
			Address: lnbot.Ptr(name),
		})
		if err != nil {
			return apiError("claiming address", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(addr)
		}

		printSuccess(fmt.Sprintf("Address claimed: %s", addr.Address))
		if addr.Cost > 0 {
			fmt.Printf("  cost: %s\n", format.Sats(addr.Cost))
		}
		return nil
	},
}

var addressTransferCmd = &cobra.Command{
	Use:   "transfer <address>",
	Short: "Transfer an address to another wallet",
	Long: `Move a Lightning address to a different wallet.

Specify the target by API key (--target-key).`,
	Example: `  lnbot address transfer alice --target-key uk_...`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]
		targetKey, _ := cmd.Flags().GetString("target-key")

		if targetKey == "" {
			return fmt.Errorf("specify --target-key <api-key>")
		}

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		if !yesFlag {
			if !confirm(fmt.Sprintf("Transfer %s to another wallet?", address)) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		result, err := w.Addresses.Transfer(context.Background(), address, &lnbot.TransferAddressParams{
			TargetWalletKey: targetKey,
		})
		if err != nil {
			return apiError("transferring address", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(result)
		}

		printSuccess(fmt.Sprintf("Transferred %s to wallet %s", result.Address, result.TransferredTo))
		return nil
	},
}

var addressDeleteCmd = &cobra.Command{
	Use:   "delete <address>",
	Short: "Delete a Lightning address",
	Long:  `Remove a Lightning address from the active wallet. This cannot be undone.`,
	Example: `  lnbot address delete alice
  lnbot address delete alice --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		address := args[0]

		if !yesFlag {
			if !confirm(fmt.Sprintf("Delete address %s?", address)) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		w, err := resolveWallet()
		if err != nil {
			return err
		}

		if err := w.Addresses.Delete(context.Background(), address); err != nil {
			return apiError("deleting address", err)
		}

		printSuccess("Address deleted")
		return nil
	},
}
