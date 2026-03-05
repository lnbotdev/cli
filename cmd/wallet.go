package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"
)

var walletCmd = &cobra.Command{
	Use:   "wallet <command>",
	Short: "Create, list, switch, and rename wallets",
	Long: `Manage wallets under your account.

All wallets share the same user key. One wallet is active at a time —
most commands operate on it by default. Use --wallet to target a
different one, or 'wallet use' to switch.`,
}

func init() {
	walletCmd.AddCommand(walletCreateCmd)
	walletCmd.AddCommand(walletListCmd)
	walletCmd.AddCommand(walletUseCmd)
	walletCmd.AddCommand(walletRenameCmd)
}

var walletCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new wallet",
	Long: `Create a new Lightning wallet on ln.bot.

Returns the wallet ID, name, and Lightning address. The wallet is
accessible via your existing user key.`,
	Example: `  lnbot wallet create
  lnbot wallet create --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		ctx := context.Background()
		ln := cfg.Client()

		wallet, err := ln.Wallets.Create(ctx)
		if err != nil {
			return apiError("creating wallet", err)
		}

		// Set as active if no active wallet
		if cfg.ActiveWalletID == "" {
			cfg.ActiveWalletID = wallet.WalletID
			if err := cfg.Save(); err != nil {
				return err
			}
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(wallet)
		}

		printSuccess("Wallet created")
		fmt.Printf("  id:       %s\n", wallet.WalletID)
		fmt.Printf("  name:     %s\n", wallet.Name)
		fmt.Printf("  address:  %s\n", wallet.Address)
		return nil
	},
}

var walletListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List wallets",
	Aliases: []string{"ls"},
	Long:    `Show all wallets under your account. The active wallet is marked with a bullet.`,
	Example: `  lnbot wallet list
  lnbot wallet list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		wallets, err := cfg.Client().Wallets.List(context.Background())
		if err != nil {
			return apiError("listing wallets", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(wallets)
		}

		if len(wallets) == 0 {
			fmt.Println("No wallets yet. Run 'lnbot wallet create' to create one.")
			return nil
		}

		for _, w := range wallets {
			marker := " "
			if w.WalletID == cfg.ActiveWalletID {
				marker = "●"
			}
			fmt.Printf("%s %s  %s\n", marker, w.Name, w.WalletID)
		}
		return nil
	},
}

var walletUseCmd = &cobra.Command{
	Use:   "use <name|id>",
	Short: "Switch the active wallet",
	Long:  `Set a wallet as active by its name or wallet ID. Subsequent commands will use this wallet by default.`,
	Example: `  lnbot wallet use agent01
  lnbot wallet use wal_7x9kQ2mR`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		target := args[0]

		wallets, err := cfg.Client().Wallets.List(context.Background())
		if err != nil {
			return apiError("listing wallets", err)
		}

		for _, w := range wallets {
			if w.WalletID == target || w.Name == target {
				cfg.ActiveWalletID = w.WalletID
				if err := cfg.Save(); err != nil {
					return err
				}
				printSuccess(fmt.Sprintf("Switched to %s (%s)", w.Name, w.WalletID))
				return nil
			}
		}

		return fmt.Errorf("wallet not found: %s\nRun 'lnbot wallet list' to see available wallets.", target)
	},
}

var walletRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename a wallet",
	Long:  `Rename a wallet on the server.`,
	Example: `  lnbot wallet rename production
  lnbot wallet rename bot-02 --wallet wal_abc`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := resolveWallet()
		if err != nil {
			return err
		}

		newName := args[0]
		if _, err := w.Update(context.Background(), &lnbot.UpdateWalletParams{
			Name: newName,
		}); err != nil {
			return apiError("renaming wallet", err)
		}

		printSuccess(fmt.Sprintf("Renamed to %s", newName))
		return nil
	},
}
