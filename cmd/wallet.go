package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
)

var walletCmd = &cobra.Command{
	Use:   "wallet <command>",
	Short: "Create, list, switch, rename, and delete wallets",
	Long: `Manage wallets in your local config.

Wallets are stored at ~/.config/lnbot/config.json (override with LNBOT_CONFIG).
One wallet is active at a time — most commands operate on it by default.
Use --wallet to target a different one, or 'wallet use' to switch.`,
}

func init() {
	walletCreateCmd.Flags().String("name", "", "wallet name (auto-generated if omitted)")

	walletDeleteCmd.Flags().Bool("force", false, "skip confirmation prompt")

	walletCmd.AddCommand(walletCreateCmd)
	walletCmd.AddCommand(walletListCmd)
	walletCmd.AddCommand(walletUseCmd)
	walletCmd.AddCommand(walletDeleteCmd)
	walletCmd.AddCommand(walletRenameCmd)
}

var walletCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new wallet",
	Long: `Create a new Lightning wallet on ln.bot.

Returns the wallet ID, API keys, Lightning address, and a 12-word recovery
passphrase. The passphrase is shown only once — save it somewhere safe.

If --name is omitted the server picks a name automatically.`,
	Example: `  lnbot wallet create --name agent01
  lnbot wallet create`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg == nil {
			var err error
			cfg, err = config.Init()
			if err != nil {
				return err
			}
		}

		name, _ := cmd.Flags().GetString("name")

		ln := config.AnonClient()
		var params *lnbot.CreateWalletParams
		if name != "" {
			params = &lnbot.CreateWalletParams{Name: lnbot.Ptr(name)}
		}
		wallet, err := ln.Wallets.Create(context.Background(), params)
		if err != nil {
			return apiError("creating wallet", err)
		}

		localName := name
		if localName == "" {
			localName = wallet.Name
		}

		cfg.Wallets[localName] = config.WalletEntry{
			ID:           wallet.WalletID,
			PrimaryKey:   wallet.PrimaryKey,
			SecondaryKey: wallet.SecondaryKey,
			Address:      wallet.Address,
		}
		if len(cfg.Wallets) == 1 || cfg.Active == "" {
			cfg.Active = localName
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(wallet)
		}

		printSuccess("Wallet created")
		fmt.Printf("  id:       %s\n", wallet.WalletID)
		fmt.Printf("  name:     %s\n", wallet.Name)
		fmt.Printf("  address:  %s\n", wallet.Address)
		fmt.Printf("  api_key:  %s\n", truncateKey(wallet.PrimaryKey))
		fmt.Println()
		printWarning("Recovery passphrase (save this — shown only once):")
		fmt.Printf("  %s\n", wallet.RecoveryPassphrase)
		return nil
	},
}

var walletListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List wallets in local config",
	Aliases: []string{"ls"},
	Long:    `Show all wallets stored in the local config file. The active wallet is marked with a bullet.`,
	Example: `  lnbot wallet list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		if len(cfg.Wallets) == 0 {
			fmt.Println("No wallets yet. Run 'lnbot wallet create' to create one.")
			return nil
		}

		for name, entry := range cfg.Wallets {
			marker := " "
			if name == cfg.Active {
				marker = "●"
			}
			fmt.Printf("%s %s  %s\n", marker, name, entry.ID)
		}
		return nil
	},
}

var walletUseCmd = &cobra.Command{
	Use:   "use <name|id>",
	Short: "Switch the active wallet",
	Long:  `Set a wallet as active by its config name or wallet ID. Subsequent commands will use this wallet by default.`,
	Example: `  lnbot wallet use agent01
  lnbot wallet use wal_7x9kQ2mR`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		target := args[0]

		if _, ok := cfg.Wallets[target]; ok {
			cfg.Active = target
			if err := cfg.Save(); err != nil {
				return err
			}
			printSuccess(fmt.Sprintf("Switched to %s", target))
			return nil
		}

		for name, entry := range cfg.Wallets {
			if entry.ID == target {
				cfg.Active = name
				if err := cfg.Save(); err != nil {
					return err
				}
				printSuccess(fmt.Sprintf("Switched to %s", name))
				return nil
			}
		}

		return fmt.Errorf("wallet not found: %s\nRun 'lnbot wallet list' to see available wallets.", target)
	},
}

var walletDeleteCmd = &cobra.Command{
	Use:   "delete [name]",
	Short: "Remove a wallet from local config",
	Long: `Remove a wallet from the local config file. This does NOT delete the
wallet on the server — it only forgets it locally.

If no name is given, the active wallet is removed.`,
	Example: `  lnbot wallet delete agent02
  lnbot wallet delete
  lnbot wallet delete agent02 --force`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		var name string
		if len(args) > 0 {
			name = args[0]
		} else {
			if cfg.Active == "" {
				return fmt.Errorf("no active wallet to delete")
			}
			name = cfg.Active
		}

		entry, ok := cfg.Wallets[name]
		if !ok {
			return fmt.Errorf("wallet %q not found in config", name)
		}

		force, _ := cmd.Flags().GetBool("force")
		if !force && !yesFlag {
			if !confirm(fmt.Sprintf("Remove '%s' (%s) from config?", name, entry.ID)) {
				fmt.Println("Cancelled.")
				return nil
			}
		}

		delete(cfg.Wallets, name)
		if cfg.Active == name {
			cfg.Active = ""
			for n := range cfg.Wallets {
				cfg.Active = n
				break
			}
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		printSuccess("Wallet removed from config")
		return nil
	},
}

var walletRenameCmd = &cobra.Command{
	Use:   "rename <new-name>",
	Short: "Rename the active wallet",
	Long:  `Rename the active wallet both on the server and in local config.`,
	Example: `  lnbot wallet rename production
  lnbot wallet rename bot-02 --wallet staging`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}
		newName := args[0]

		ln, entry, oldName, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		_, err = ln.Wallets.Update(context.Background(), &lnbot.UpdateWalletParams{
			Name: newName,
		})
		if err != nil {
			return apiError("renaming wallet", err)
		}

		cfg.Wallets[newName] = *entry
		delete(cfg.Wallets, oldName)
		if cfg.Active == oldName {
			cfg.Active = newName
		}
		if err := cfg.Save(); err != nil {
			return err
		}

		printSuccess(fmt.Sprintf("Renamed to %s", newName))
		return nil
	},
}
