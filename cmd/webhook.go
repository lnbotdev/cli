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

var webhookCmd = &cobra.Command{
	Use:   "webhook <command>",
	Short: "Manage webhook endpoints",
	Long: `Register, list, and delete webhook endpoints.

Webhooks receive real-time HTTP POST notifications for wallet events
(payments received, invoices settled, etc).`,
}

func init() {
	webhookCreateCmd.Flags().String("url", "", "webhook endpoint URL (required)")
	webhookCreateCmd.MarkFlagRequired("url")

	webhookCmd.AddCommand(webhookCreateCmd)
	webhookCmd.AddCommand(webhookListCmd)
	webhookCmd.AddCommand(webhookDeleteCmd)
}

var webhookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Register a webhook endpoint",
	Long: `Register a new webhook URL. The server will send a signing secret
that you use to verify payloads. The secret is shown once.`,
	Example: `  lnbot webhook create --url https://myapp.com/hooks/lnbot
  lnbot webhook create --url https://example.com/hook --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		url, _ := cmd.Flags().GetString("url")

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		hook, err := ln.Webhooks.Create(context.Background(), &lnbot.CreateWebhookParams{
			URL: url,
		})
		if err != nil {
			return apiError("creating webhook", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(hook)
		}

		printSuccess("Webhook created")
		fmt.Printf("  id:     %s\n", hook.ID)
		fmt.Printf("  url:    %s\n", hook.URL)
		fmt.Printf("  secret: %s\n", hook.Secret)
		fmt.Println()
		fmt.Println("  Save the secret — it won't be shown again.")
		return nil
	},
}

var webhookListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List webhook endpoints",
	Aliases: []string{"ls"},
	Long:    `Show all registered webhooks for the active wallet.`,
	Example: `  lnbot webhook list
  lnbot webhook list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		hooks, err := ln.Webhooks.List(context.Background())
		if err != nil {
			return apiError("listing webhooks", err)
		}

		if jsonFlag {
			return json.NewEncoder(os.Stdout).Encode(hooks)
		}

		if len(hooks) == 0 {
			fmt.Println("No webhooks yet.")
			return nil
		}

		for _, h := range hooks {
			status := "active"
			if !h.Active {
				status = "inactive"
			}
			fmt.Printf("  %s  %-8s  %s  %s\n", h.ID, status, h.URL, format.TimeAgo(h.CreatedAt))
		}
		return nil
	},
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a webhook endpoint",
	Long:  `Remove a webhook endpoint. It will stop receiving events immediately.`,
	Example: `  lnbot webhook delete whk_9xMn2
  lnbot webhook delete whk_9xMn2 --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireConfig(); err != nil {
			return err
		}

		id := args[0]

		ln, _, _, err := cfg.Client(walletFlag)
		if err != nil {
			return err
		}

		if err := ln.Webhooks.Delete(context.Background(), id); err != nil {
			return apiError("deleting webhook", err)
		}

		printSuccess(fmt.Sprintf("Webhook %s deleted", id))
		return nil
	},
}
