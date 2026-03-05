//go:build integration

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lnbotdev/cli/internal/config"
)

// Integration tests — run the CLI commands against the live API.
//
// Required environment variables:
//   LNBOT_USER_KEY=uk_...   # user key that owns the prefunded wallet
//   LNBOT_WALLET_ID=wal_... # prefunded wallet ID
//
// Run:
//   go test -tags=integration -run TestInteg -v -count=1 -timeout=120s

var (
	integConfigPath string
	fundedUserKey   string
	fundedWalletID  string
)

func TestMain(m *testing.M) {
	fundedUserKey = os.Getenv("LNBOT_USER_KEY")
	fundedWalletID = os.Getenv("LNBOT_WALLET_ID")
	if fundedUserKey == "" || fundedWalletID == "" {
		fmt.Println("SKIP: LNBOT_USER_KEY and LNBOT_WALLET_ID required")
		os.Exit(0)
	}

	os.Setenv("LNBOT_NO_UPDATE_CHECK", "1")
	os.Exit(m.Run())
}

func integSetup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	integConfigPath = filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", integConfigPath)
}

func integSetupWithConfig(t *testing.T, c *config.Config) {
	t.Helper()
	integSetup(t)
	data, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile(integConfigPath, data, 0o600)
}

// ── Init ─────────────────────────────────────────────────

func TestInteg_Init(t *testing.T) {
	integSetup(t)

	stdout, _, err := executeCmd("init")
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
	if !strings.Contains(stdout, "Account and wallet created") {
		t.Errorf("expected success message, got %q", stdout)
	}
	if !strings.Contains(stdout, "wal_") {
		t.Errorf("expected wallet ID in output, got %q", stdout)
	}
	if !strings.Contains(stdout, "@ln.bot") {
		t.Errorf("expected Lightning address in output, got %q", stdout)
	}

	// Verify config file was created
	data, err := os.ReadFile(integConfigPath)
	if err != nil {
		t.Fatalf("config file not created: %v", err)
	}
	var saved config.Config
	json.Unmarshal(data, &saved)
	if !strings.HasPrefix(saved.PrimaryKey, "uk_") {
		t.Errorf("PrimaryKey = %q, want uk_ prefix", saved.PrimaryKey)
	}
	if !strings.HasPrefix(saved.ActiveWalletID, "wal_") {
		t.Errorf("ActiveWalletID = %q, want wal_ prefix", saved.ActiveWalletID)
	}
}

func TestInteg_Init_JSON(t *testing.T) {
	integSetup(t)

	stdout, _, err := executeCmd("init", "--json")
	if err != nil {
		t.Fatalf("init --json failed: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}
	if !strings.HasPrefix(result["primary_key"], "uk_") {
		t.Errorf("primary_key = %q", result["primary_key"])
	}
	if !strings.HasPrefix(result["wallet_id"], "wal_") {
		t.Errorf("wallet_id = %q", result["wallet_id"])
	}
	words := strings.Fields(result["recovery_passphrase"])
	if len(words) != 12 {
		t.Errorf("recovery_passphrase has %d words, want 12", len(words))
	}
}

func TestInteg_Init_AlreadyExists(t *testing.T) {
	integSetup(t)

	executeCmd("init")
	stdout, _, err := executeCmd("init")
	if err != nil {
		t.Fatalf("second init failed: %v", err)
	}
	if !strings.Contains(stdout, "Already initialized") {
		t.Errorf("expected 'Already initialized', got %q", stdout)
	}
}

// ── Wallet ───────────────────────────────────────────────

func TestInteg_Wallet_CreateAndList(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	// Create a second wallet
	stdout, _, err := executeCmd("wallet", "create")
	if err != nil {
		t.Fatalf("wallet create failed: %v", err)
	}
	if !strings.Contains(stdout, "Wallet created") {
		t.Errorf("expected 'Wallet created', got %q", stdout)
	}

	// List should show at least 2 wallets
	stdout, _, err = executeCmd("wallet", "list")
	if err != nil {
		t.Fatalf("wallet list failed: %v", err)
	}
	if !strings.Contains(stdout, "●") {
		t.Errorf("expected active marker in output, got %q", stdout)
	}
	if !strings.Contains(stdout, "wal_") {
		t.Errorf("expected wallet IDs in output, got %q", stdout)
	}
}

func TestInteg_Wallet_ListJSON(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	stdout, _, err := executeCmd("wallet", "list", "--json")
	if err != nil {
		t.Fatalf("wallet list --json failed: %v", err)
	}
	var wallets []map[string]any
	if err := json.Unmarshal([]byte(stdout), &wallets); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(wallets) < 1 {
		t.Error("expected at least 1 wallet")
	}
}

func TestInteg_Wallet_UseByName(t *testing.T) {
	integSetup(t)
	executeCmd("init")
	executeCmd("wallet", "create")

	// Get wallet list to find names
	stdout, _, _ := executeCmd("wallet", "list", "--json")
	var wallets []map[string]any
	json.Unmarshal([]byte(stdout), &wallets)
	if len(wallets) < 2 {
		t.Skip("need at least 2 wallets")
	}

	// Switch to the second wallet by name
	name := wallets[1]["name"].(string)
	stdout, _, err := executeCmd("wallet", "use", name)
	if err != nil {
		t.Fatalf("wallet use failed: %v", err)
	}
	if !strings.Contains(stdout, "Switched to") {
		t.Errorf("expected switch message, got %q", stdout)
	}
}

func TestInteg_Wallet_UseByID(t *testing.T) {
	integSetup(t)
	executeCmd("init")
	executeCmd("wallet", "create")

	stdout, _, _ := executeCmd("wallet", "list", "--json")
	var wallets []map[string]any
	json.Unmarshal([]byte(stdout), &wallets)
	if len(wallets) < 2 {
		t.Skip("need at least 2 wallets")
	}

	id := wallets[1]["walletId"].(string)
	stdout, _, err := executeCmd("wallet", "use", id)
	if err != nil {
		t.Fatalf("wallet use failed: %v", err)
	}
	if !strings.Contains(stdout, "Switched to") {
		t.Errorf("expected switch message, got %q", stdout)
	}
}

func TestInteg_Wallet_UseNotFound(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	_, _, err := executeCmd("wallet", "use", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown wallet")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInteg_Wallet_Rename(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	newName := fmt.Sprintf("cli-test-%d", os.Getpid())
	stdout, _, err := executeCmd("wallet", "rename", newName)
	if err != nil {
		t.Fatalf("wallet rename failed: %v", err)
	}
	if !strings.Contains(stdout, newName) {
		t.Errorf("expected new name in output, got %q", stdout)
	}

	// Verify via list
	stdout, _, _ = executeCmd("wallet", "list")
	if !strings.Contains(stdout, newName) {
		t.Errorf("wallet list should contain new name %q, got %q", newName, stdout)
	}
}

// ── Balance ──────────────────────────────────────────────

func TestInteg_Balance(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("balance")
	if err != nil {
		t.Fatalf("balance failed: %v", err)
	}
	if !strings.Contains(stdout, "balance:") {
		t.Errorf("expected 'balance:' in output, got %q", stdout)
	}
	if !strings.Contains(stdout, "available:") {
		t.Errorf("expected 'available:' in output, got %q", stdout)
	}
}

func TestInteg_Balance_JSON(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("balance", "--json")
	if err != nil {
		t.Fatalf("balance --json failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result["balance"]; !ok {
		t.Error("missing 'balance' field")
	}
	if _, ok := result["available"]; !ok {
		t.Error("missing 'available' field")
	}
}

// ── Status ───────────────────────────────────────────────

func TestInteg_Status(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("status")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if !strings.Contains(stdout, "connected") {
		t.Errorf("expected 'connected' in output, got %q", stdout)
	}
	if !strings.Contains(stdout, fundedWalletID) {
		t.Errorf("expected wallet ID in output, got %q", stdout)
	}
}

func TestInteg_Status_JSON(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("status", "--json")
	if err != nil {
		t.Fatalf("status --json failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["walletId"] != fundedWalletID {
		t.Errorf("walletId = %v, want %q", result["walletId"], fundedWalletID)
	}
}

// ── Whoami ───────────────────────────────────────────────

func TestInteg_Whoami(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("whoami")
	if err != nil {
		t.Fatalf("whoami failed: %v", err)
	}
	if !strings.Contains(stdout, fundedWalletID) {
		t.Errorf("expected wallet ID, got %q", stdout)
	}
}

func TestInteg_Whoami_JSON(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("whoami", "--json")
	if err != nil {
		t.Fatalf("whoami --json failed: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["wallet_id"] != fundedWalletID {
		t.Errorf("wallet_id = %q, want %q", result["wallet_id"], fundedWalletID)
	}
}

// ── Key ──────────────────────────────────────────────────

func TestInteg_Key_Show(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		SecondaryKey:   "uk_secondary_test",
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("key", "show")
	if err != nil {
		t.Fatalf("key show failed: %v", err)
	}
	if !strings.Contains(stdout, fundedUserKey) {
		t.Errorf("expected user key in output, got %q", stdout)
	}
}

// Key rotate is tested on a fresh account to avoid breaking the funded wallet
func TestInteg_Key_Rotate(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	// Read the config to get the key before rotation
	data, _ := os.ReadFile(integConfigPath)
	var before config.Config
	json.Unmarshal(data, &before)

	stdout, _, err := executeCmd("key", "rotate", "1", "--yes")
	if err != nil {
		t.Fatalf("key rotate failed: %v", err)
	}
	if !strings.Contains(stdout, "key rotated") {
		t.Errorf("expected 'key rotated', got %q", stdout)
	}

	// Verify config updated
	data, _ = os.ReadFile(integConfigPath)
	var after config.Config
	json.Unmarshal(data, &after)
	if after.SecondaryKey == before.SecondaryKey {
		t.Error("secondary key should have changed after rotation")
	}
}

// ── Address ──────────────────────────────────────────────

func TestInteg_Address_List(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("address", "list")
	if err != nil {
		t.Fatalf("address list failed: %v", err)
	}
	if !strings.Contains(stdout, "@ln.bot") {
		t.Errorf("expected address in output, got %q", stdout)
	}
}

func TestInteg_Address_BuyAndDelete(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	name := fmt.Sprintf("clitest%d", os.Getpid()%100000)

	stdout, _, err := executeCmd("address", "buy", name, "--yes")
	if err != nil {
		t.Fatalf("address buy failed: %v", err)
	}
	if !strings.Contains(stdout, name) {
		t.Errorf("expected address name in output, got %q", stdout)
	}

	// Delete
	stdout, _, err = executeCmd("address", "delete", name, "--yes")
	if err != nil {
		t.Fatalf("address delete failed: %v", err)
	}
	if !strings.Contains(stdout, "deleted") {
		t.Errorf("expected 'deleted', got %q", stdout)
	}
}

// ── Invoice ──────────────────────────────────────────────

func TestInteg_Invoice_Create_NoWait(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("invoice", "create", "--amount", "100", "--memo", "cli-test", "--no-wait")
	if err != nil {
		t.Fatalf("invoice create failed: %v", err)
	}
	if !strings.Contains(stdout, "lnbc") {
		t.Errorf("expected bolt11 in output, got %q", stdout)
	}
	if !strings.Contains(stdout, "100 sats") {
		t.Errorf("expected amount in output, got %q", stdout)
	}
}

func TestInteg_Invoice_Create_JSON(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("invoice", "create", "--amount", "50", "--no-wait", "--json")
	if err != nil {
		t.Fatalf("invoice create --json failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["status"] != "pending" {
		t.Errorf("status = %v, want pending", result["status"])
	}
	bolt11, ok := result["bolt11"].(string)
	if !ok || !strings.HasPrefix(bolt11, "lnbc") {
		t.Errorf("bolt11 = %v, want lnbc prefix", result["bolt11"])
	}
}

func TestInteg_Invoice_List(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	// Create one first
	executeCmd("invoice", "create", "--amount", "10", "--no-wait")

	stdout, _, err := executeCmd("invoice", "list")
	if err != nil {
		t.Fatalf("invoice list failed: %v", err)
	}
	if !strings.Contains(stdout, "pending") && !strings.Contains(stdout, "settled") && !strings.Contains(stdout, "expired") {
		t.Errorf("expected invoice status in output, got %q", stdout)
	}
}

// ── Payment ──────────────────────────────────────────────

func TestInteg_Pay_And_PaymentList(t *testing.T) {
	// Use a fresh account as receiver, funded wallet as sender
	integSetup(t)
	executeCmd("init")

	// Read fresh config to get the new wallet address
	data, _ := os.ReadFile(integConfigPath)
	var freshCfg config.Config
	json.Unmarshal(data, &freshCfg)

	// Get fresh wallet's address via API
	freshClient := freshCfg.Client()
	ctx := context.Background()
	addrs, err := freshClient.Wallet(freshCfg.ActiveWalletID).Addresses.List(ctx)
	if err != nil || len(addrs) == 0 {
		t.Skip("cannot get fresh wallet address")
	}

	// Switch to funded wallet for paying
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	target := addrs[0].Address
	stdout, _, err := executeCmd("pay", target, "--amount", "100", "--yes", "--no-wait")
	if err != nil {
		t.Fatalf("pay failed: %v", err)
	}
	if !strings.Contains(stdout, "status:") {
		t.Errorf("expected status in output, got %q", stdout)
	}

	// List payments
	stdout, _, err = executeCmd("payment", "list", "--limit", "1")
	if err != nil {
		t.Fatalf("payment list failed: %v", err)
	}
	if !strings.Contains(stdout, "sats") {
		t.Errorf("expected sats in output, got %q", stdout)
	}
}

func TestInteg_Pay_JSON(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	data, _ := os.ReadFile(integConfigPath)
	var freshCfg config.Config
	json.Unmarshal(data, &freshCfg)

	freshClient := freshCfg.Client()
	ctx := context.Background()
	addrs, err := freshClient.Wallet(freshCfg.ActiveWalletID).Addresses.List(ctx)
	if err != nil || len(addrs) == 0 {
		t.Skip("cannot get fresh wallet address")
	}

	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	target := addrs[0].Address
	stdout, _, err := executeCmd("pay", target, "--amount", "100", "--yes", "--no-wait", "--json")
	if err != nil {
		t.Fatalf("pay --json failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if _, ok := result["number"]; !ok {
		t.Error("missing 'number' field in payment JSON")
	}
}

// ── Transactions ─────────────────────────────────────────

func TestInteg_Transactions(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("transactions", "--limit", "5")
	if err != nil {
		t.Fatalf("transactions failed: %v", err)
	}
	// Funded wallet should have some transactions
	if strings.Contains(stdout, "No transactions") {
		t.Skip("funded wallet has no transactions")
	}
	if !strings.Contains(stdout, "sats") {
		t.Errorf("expected sats in output, got %q", stdout)
	}
}

func TestInteg_Transactions_JSON(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("transactions", "--limit", "5", "--json")
	if err != nil {
		t.Fatalf("transactions --json failed: %v", err)
	}
	var result []map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

// ── Webhook ──────────────────────────────────────────────

func TestInteg_Webhook_CRUD(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	// Create
	stdout, _, err := executeCmd("webhook", "create", "--url", "https://example.com/cli-test")
	if err != nil {
		t.Fatalf("webhook create failed: %v", err)
	}
	if !strings.Contains(stdout, "Webhook created") {
		t.Errorf("expected 'Webhook created', got %q", stdout)
	}
	if !strings.Contains(stdout, "secret:") {
		t.Errorf("expected secret in output, got %q", stdout)
	}

	// List
	stdout, _, err = executeCmd("webhook", "list")
	if err != nil {
		t.Fatalf("webhook list failed: %v", err)
	}
	if !strings.Contains(stdout, "example.com/cli-test") {
		t.Errorf("expected webhook URL in list, got %q", stdout)
	}

	// Get webhook ID from JSON list
	stdout, _, _ = executeCmd("webhook", "list", "--json")
	var hooks []map[string]any
	json.Unmarshal([]byte(stdout), &hooks)
	if len(hooks) == 0 {
		t.Fatal("no webhooks in list")
	}
	hookID := hooks[0]["id"].(string)

	// Delete
	stdout, _, err = executeCmd("webhook", "delete", hookID)
	if err != nil {
		t.Fatalf("webhook delete failed: %v", err)
	}
	if !strings.Contains(stdout, "deleted") {
		t.Errorf("expected 'deleted', got %q", stdout)
	}
}

// ── MCP ──────────────────────────────────────────────────

func TestInteg_MCP_Config(t *testing.T) {
	integSetupWithConfig(t, &config.Config{
		PrimaryKey:     fundedUserKey,
		ActiveWalletID: fundedWalletID,
	})

	stdout, _, err := executeCmd("mcp", "config", "--remote")
	if err != nil {
		t.Fatalf("mcp config --remote failed: %v", err)
	}
	if !strings.Contains(stdout, fundedUserKey) {
		t.Errorf("expected user key in output, got %q", stdout)
	}
	expectedURL := fmt.Sprintf("api.ln.bot/v1/wallets/%s/mcp", fundedWalletID)
	if !strings.Contains(stdout, expectedURL) {
		t.Errorf("expected MCP URL %q in output, got %q", expectedURL, stdout)
	}
}

// ── Backup & Restore ─────────────────────────────────────

func TestInteg_Backup_Recovery(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	stdout, _, err := executeCmd("backup", "recovery")
	if err != nil {
		t.Fatalf("backup recovery failed: %v", err)
	}
	if !strings.Contains(stdout, "Recovery passphrase") {
		t.Errorf("expected passphrase header, got %q", stdout)
	}
}

func TestInteg_Backup_Recovery_JSON(t *testing.T) {
	integSetup(t)
	executeCmd("init")

	stdout, _, err := executeCmd("backup", "recovery", "--json")
	if err != nil {
		t.Fatalf("backup recovery --json failed: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	words := strings.Fields(result["passphrase"])
	if len(words) != 12 {
		t.Errorf("passphrase has %d words, want 12", len(words))
	}
}

func TestInteg_Restore_Recovery(t *testing.T) {
	// Create account, get passphrase, then restore
	integSetup(t)
	executeCmd("init")

	// Get recovery passphrase
	stdout, _, _ := executeCmd("backup", "recovery", "--json")
	var backupResult map[string]string
	json.Unmarshal([]byte(stdout), &backupResult)
	passphrase := backupResult["passphrase"]
	if passphrase == "" {
		t.Fatal("could not get recovery passphrase")
	}

	// Read original config
	data, _ := os.ReadFile(integConfigPath)
	var originalCfg config.Config
	json.Unmarshal(data, &originalCfg)

	// Delete config and restore
	os.Remove(integConfigPath)

	stdout, _, err := executeCmd("restore", "recovery", "--passphrase", passphrase)
	if err != nil {
		t.Fatalf("restore recovery failed: %v", err)
	}
	if !strings.Contains(stdout, "Account restored") {
		t.Errorf("expected 'Account restored', got %q", stdout)
	}

	// Verify config was recreated with new keys
	data, _ = os.ReadFile(integConfigPath)
	var restoredCfg config.Config
	json.Unmarshal(data, &restoredCfg)
	if !strings.HasPrefix(restoredCfg.PrimaryKey, "uk_") {
		t.Errorf("PrimaryKey = %q, want uk_ prefix", restoredCfg.PrimaryKey)
	}
	if restoredCfg.PrimaryKey == originalCfg.PrimaryKey {
		t.Error("PrimaryKey should have changed after restore")
	}
}

// ── Wallet flag ──────────────────────────────────────────

func TestInteg_WalletFlag_ByID(t *testing.T) {
	integSetup(t)
	executeCmd("init")
	executeCmd("wallet", "create")

	// Get second wallet ID
	stdout, _, _ := executeCmd("wallet", "list", "--json")
	var wallets []map[string]any
	json.Unmarshal([]byte(stdout), &wallets)
	if len(wallets) < 2 {
		t.Skip("need at least 2 wallets")
	}

	// Find the non-active wallet
	data, _ := os.ReadFile(integConfigPath)
	var cfg config.Config
	json.Unmarshal(data, &cfg)

	var otherID string
	for _, w := range wallets {
		id := w["walletId"].(string)
		if id != cfg.ActiveWalletID {
			otherID = id
			break
		}
	}

	// Balance with --wallet flag should work
	stdout, _, err := executeCmd("balance", "--wallet", otherID)
	if err != nil {
		t.Fatalf("balance --wallet failed: %v", err)
	}
	if !strings.Contains(stdout, "balance:") {
		t.Errorf("expected balance output, got %q", stdout)
	}
}

// ── Error cases ──────────────────────────────────────────

func TestInteg_NoConfig_Commands(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LNBOT_CONFIG", filepath.Join(dir, "nope.json"))
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "1")

	commands := [][]string{
		{"balance"},
		{"status"},
		{"whoami"},
		{"invoice", "list"},
		{"payment", "list"},
		{"transactions"},
		{"address", "list"},
		{"webhook", "list"},
		{"key", "show"},
		{"wallet", "list"},
	}
	for _, args := range commands {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			_, _, err := executeCmd(args...)
			if err == nil {
				t.Errorf("expected error for %v without config", args)
			}
			if !strings.Contains(err.Error(), "no config found") {
				t.Errorf("unexpected error for %v: %v", args, err)
			}
		})
	}
}
