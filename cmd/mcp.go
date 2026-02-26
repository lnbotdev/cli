package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp <command>",
	Short: "MCP server config for AI agents",
	Long: `Generate configuration for the Model Context Protocol (MCP) server.

MCP lets AI agents (Claude, Cursor, etc.) use your wallet. The 'config'
command prints JSON you paste into your MCP client settings.`,
}

func init() {
	mcpConfigCmd.Flags().Bool("remote", false, "generate remote config (required for now)")

	mcpCmd.AddCommand(mcpConfigCmd)
	mcpCmd.AddCommand(mcpServeCmd)
}

var mcpConfigCmd = &cobra.Command{
	Use:   "config --remote",
	Short: "Print MCP server configuration JSON",
	Long: `Print the JSON config block to add to your MCP client (Claude Desktop,
Cursor, etc).

Currently only --remote is supported, which uses the hosted endpoint.
Local stdio mode is coming soon.`,
	Example: `  lnbot mcp config --remote
  lnbot mcp config --remote --wallet agent02`,
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := cmd.Flags().GetBool("remote")

		if !remote {
			fmt.Println("Local MCP server is not available yet.")
			fmt.Println()
			fmt.Println("  Use --remote for the hosted endpoint:")
			fmt.Println("    lnbot mcp config --remote")
			return nil
		}

		if err := requireConfig(); err != nil {
			return err
		}
		entry, _, err := cfg.ResolveWallet(walletFlag)
		if err != nil {
			return err
		}

		config := map[string]any{
			"mcpServers": map[string]any{
				"lnbot": map[string]any{
					"type": "url",
					"url":  "https://api.ln.bot/mcp",
					"headers": map[string]string{
						"Authorization": "Bearer " + entry.PrimaryKey,
					},
				},
			},
		}

		fmt.Println("Add to your MCP client config:")
		fmt.Println()

		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(config)
	},
}

var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start local MCP server (coming soon)",
	Long:  `Start a local MCP server over stdio. This is not available yet.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Local MCP server is not available yet.")
		fmt.Println()
		fmt.Println("  Use the remote endpoint instead:")
		fmt.Println("    lnbot mcp config --remote")
		return nil
	},
}
