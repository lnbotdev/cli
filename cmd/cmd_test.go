package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	lnbot "github.com/lnbotdev/go-sdk"

	"github.com/lnbotdev/cli/internal/config"
)

// ---------------------------------------------------------------------------
// Test infrastructure
// ---------------------------------------------------------------------------

func resetState() {
	cfg = nil
	walletFlag = ""
	jsonFlag = false
	yesFlag = false
}

func executeCmd(args ...string) (stdout, stderr string, err error) {
	resetState()

	// Capture os.Stdout
	oldStdout := os.Stdout
	rOut, wOut, _ := os.Pipe()
	os.Stdout = wOut

	// Capture os.Stderr
	oldStderr := os.Stderr
	rErr, wErr, _ := os.Pipe()
	os.Stderr = wErr

	rootCmd.SetArgs(args)
	err = rootCmd.Execute()

	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	outBytes, _ := io.ReadAll(rOut)
	errBytes, _ := io.ReadAll(rErr)

	return string(outBytes), string(errBytes), err
}

func setupConfig(t *testing.T, c *config.Config) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", p)
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "1")
	if c != nil {
		data, _ := json.MarshalIndent(c, "", "  ")
		os.WriteFile(p, data, 0o600)
	}
	return p
}

func setupNoConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("LNBOT_CONFIG", filepath.Join(dir, "nonexistent", "config.json"))
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "1")
}

func testConfig() *config.Config {
	return &config.Config{
		Active: "main",
		Wallets: map[string]config.WalletEntry{
			"main": {
				ID:           "wal_main123",
				PrimaryKey:   "key_primary_abcdefghijklmnop",
				SecondaryKey: "key_secondary_1234567890abcdef",
				Address:      "main@ln.bot",
			},
			"secondary": {
				ID:           "wal_sec456",
				PrimaryKey:   "key_sec_primary_xyz",
				SecondaryKey: "key_sec_secondary_xyz",
				Address:      "secondary@ln.bot",
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Helper function tests
// ---------------------------------------------------------------------------

func TestTruncateKey(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want string
	}{
		{"short key", "abc123", "abc123"},
		{"exactly 16", "1234567890123456", "1234567890123456"},
		{"long key", "key_primary_abcdefghijklmnop", "key_primary_...mnop"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := truncateKey(tt.key); got != tt.want {
				t.Errorf("truncateKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestApiError_GenericError(t *testing.T) {
	err := apiError("testing", errors.New("connection refused"))
	if !strings.Contains(err.Error(), "testing") {
		t.Errorf("error should contain action: %v", err)
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error should contain original message: %v", err)
	}
}

func TestApiError_APIError(t *testing.T) {
	apiErr := &lnbot.APIError{StatusCode: 400, Message: "bad request"}
	err := apiError("creating invoice", apiErr)
	if !strings.Contains(err.Error(), "creating invoice") {
		t.Errorf("error should contain action: %v", err)
	}
	if !strings.Contains(err.Error(), "bad request") {
		t.Errorf("error should contain API message: %v", err)
	}
}

func TestRequireConfig_Nil(t *testing.T) {
	cfg = nil
	err := requireConfig()
	if err == nil {
		t.Fatal("expected error for nil config")
	}
	if !strings.Contains(err.Error(), "no config found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRequireConfig_NonNil(t *testing.T) {
	cfg = &config.Config{}
	err := requireConfig()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	cfg = nil
}

// ---------------------------------------------------------------------------
// Version command
// ---------------------------------------------------------------------------

func TestVersion(t *testing.T) {
	setupNoConfig(t)
	stdout, _, err := executeCmd("version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, version) {
		t.Errorf("output %q should contain version %q", stdout, version)
	}
}

// ---------------------------------------------------------------------------
// Init command
// ---------------------------------------------------------------------------

func TestInit_New(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", p)
	t.Setenv("LNBOT_NO_UPDATE_CHECK", "1")

	stdout, _, err := executeCmd("init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Config created") {
		t.Errorf("output should contain 'Config created', got %q", stdout)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestInit_Existing(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("init")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Config already exists") {
		t.Errorf("output should contain 'Config already exists', got %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// Wallet list
// ---------------------------------------------------------------------------

func TestWalletList_Empty(t *testing.T) {
	setupConfig(t, &config.Config{Wallets: map[string]config.WalletEntry{}})

	stdout, _, err := executeCmd("wallet", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "No wallets yet") {
		t.Errorf("output should contain 'No wallets yet', got %q", stdout)
	}
}

func TestWalletList_WithWallets(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("wallet", "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "main") {
		t.Errorf("output should contain wallet name 'main', got %q", stdout)
	}
	if !strings.Contains(stdout, "secondary") {
		t.Errorf("output should contain wallet name 'secondary', got %q", stdout)
	}
	if !strings.Contains(stdout, "●") {
		t.Errorf("output should contain active marker ●, got %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// Wallet use
// ---------------------------------------------------------------------------

func TestWalletUse_ByName(t *testing.T) {
	p := setupConfig(t, testConfig())

	stdout, _, err := executeCmd("wallet", "use", "secondary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Switched to secondary") {
		t.Errorf("output should contain switch message, got %q", stdout)
	}

	// Verify persisted
	data, _ := os.ReadFile(p)
	var saved config.Config
	json.Unmarshal(data, &saved)
	if saved.Active != "secondary" {
		t.Errorf("saved active = %q, want %q", saved.Active, "secondary")
	}
}

func TestWalletUse_ByID(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("wallet", "use", "wal_sec456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Switched to secondary") {
		t.Errorf("output should contain switch message, got %q", stdout)
	}
}

func TestWalletUse_NotFound(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("wallet", "use", "nonexistent")
	if err == nil {
		t.Fatal("expected error for not found wallet")
	}
	if !strings.Contains(err.Error(), "wallet not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Wallet delete
// ---------------------------------------------------------------------------

func TestWalletDelete_WithForce(t *testing.T) {
	p := setupConfig(t, testConfig())

	stdout, _, err := executeCmd("wallet", "delete", "secondary", "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Wallet removed") {
		t.Errorf("output should contain 'Wallet removed', got %q", stdout)
	}

	data, _ := os.ReadFile(p)
	var saved config.Config
	json.Unmarshal(data, &saved)
	if _, ok := saved.Wallets["secondary"]; ok {
		t.Error("secondary wallet should have been removed")
	}
}

func TestWalletDelete_ActiveWithYes(t *testing.T) {
	p := setupConfig(t, testConfig())

	stdout, _, err := executeCmd("wallet", "delete", "main", "--yes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "Wallet removed") {
		t.Errorf("output should contain 'Wallet removed', got %q", stdout)
	}

	data, _ := os.ReadFile(p)
	var saved config.Config
	json.Unmarshal(data, &saved)
	if _, ok := saved.Wallets["main"]; ok {
		t.Error("main wallet should have been removed")
	}
	// Active should switch to remaining wallet or be empty
	if saved.Active == "main" {
		t.Error("active should not still be 'main' after deletion")
	}
}

func TestWalletDelete_NotFound(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("wallet", "delete", "nonexistent", "--force")
	if err == nil {
		t.Fatal("expected error for not found wallet")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Key show
// ---------------------------------------------------------------------------

func TestKeyShow(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("key", "show")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "primary") {
		t.Errorf("output should contain 'primary', got %q", stdout)
	}
	if !strings.Contains(stdout, "key_primary_abcdefghijklmnop") {
		t.Errorf("output should contain primary key value, got %q", stdout)
	}
	if !strings.Contains(stdout, "secondary") {
		t.Errorf("output should contain 'secondary', got %q", stdout)
	}
}

func TestKeyShow_JSON(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("key", "show", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["primary_key"] != "key_primary_abcdefghijklmnop" {
		t.Errorf("primary_key = %q, want %q", result["primary_key"], "key_primary_abcdefghijklmnop")
	}
	if result["secondary_key"] != "key_secondary_1234567890abcdef" {
		t.Errorf("secondary_key = %q, want %q", result["secondary_key"], "key_secondary_1234567890abcdef")
	}
}

func TestKeyShow_Wallet(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("key", "show", "--wallet", "secondary")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "key_sec_primary_xyz") {
		t.Errorf("output should contain secondary wallet's key, got %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// Whoami (JSON path — no API call)
// ---------------------------------------------------------------------------

func TestWhoami_JSON(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("whoami", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result["wallet"] != "wal_main123" {
		t.Errorf("wallet = %q, want %q", result["wallet"], "wal_main123")
	}
	if result["name"] != "main" {
		t.Errorf("name = %q, want %q", result["name"], "main")
	}
	if _, ok := result["api_key"]; !ok {
		t.Error("api_key field missing")
	}
}

// ---------------------------------------------------------------------------
// MCP
// ---------------------------------------------------------------------------

func TestMcpConfig_NoRemote(t *testing.T) {
	setupNoConfig(t)

	stdout, _, err := executeCmd("mcp", "config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "not available yet") {
		t.Errorf("output should contain 'not available yet', got %q", stdout)
	}
}

func TestMcpConfig_Remote(t *testing.T) {
	setupConfig(t, testConfig())

	stdout, _, err := executeCmd("mcp", "config", "--remote")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "key_primary_abcdefghijklmnop") {
		t.Errorf("output should contain API key, got %q", stdout)
	}
	if !strings.Contains(stdout, "api.ln.bot/mcp") {
		t.Errorf("output should contain MCP URL, got %q", stdout)
	}
}

func TestMcpServe(t *testing.T) {
	setupNoConfig(t)

	stdout, _, err := executeCmd("mcp", "serve")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout, "not available yet") {
		t.Errorf("output should contain 'not available yet', got %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// Completion
// ---------------------------------------------------------------------------

func TestCompletion_Bash(t *testing.T) {
	setupNoConfig(t)

	stdout, _, err := executeCmd("completion", "bash")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stdout) == 0 {
		t.Error("completion output should not be empty")
	}
	if !strings.Contains(stdout, "bash") {
		t.Errorf("output should contain bash-related content, got %q", stdout[:min(100, len(stdout))])
	}
}

// ---------------------------------------------------------------------------
// Validation-only tests (API commands, no actual API calls)
// ---------------------------------------------------------------------------

func TestBalance_NoConfig(t *testing.T) {
	setupNoConfig(t)

	_, _, err := executeCmd("balance")
	if err == nil {
		t.Fatal("expected error without config")
	}
	if !strings.Contains(err.Error(), "no config found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvoiceCreate_NoAmount(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("invoice", "create")
	if err == nil {
		t.Fatal("expected error without --amount")
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvoiceCreate_NegativeAmount(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("invoice", "create", "--amount", "-10")
	if err == nil {
		t.Fatal("expected error for negative amount")
	}
	if !strings.Contains(err.Error(), "positive integer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPay_UnrecognizedTarget(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("pay", "randomstring", "--yes")
	if err == nil {
		t.Fatal("expected error for unrecognized target")
	}
	if !strings.Contains(err.Error(), "unrecognized target") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPay_AddressNoAmount(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("pay", "user@domain.com", "--yes")
	if err == nil {
		t.Fatal("expected error for address without --amount")
	}
	if !strings.Contains(err.Error(), "--amount is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestPay_LNURLNoAmount(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("pay", "lnurl1dp68gurn8ghj7", "--yes")
	if err == nil {
		t.Fatal("expected error for LNURL without --amount")
	}
	if !strings.Contains(err.Error(), "--amount is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKeyRotate_InvalidSlot(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("key", "rotate", "3", "--yes")
	if err == nil {
		t.Fatal("expected error for invalid slot")
	}
	if !strings.Contains(err.Error(), "slot must be 0") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBackupPasskey(t *testing.T) {
	setupNoConfig(t)

	_, _, err := executeCmd("backup", "passkey")
	if err == nil {
		t.Fatal("expected error for passkey backup in CLI")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRestorePasskey(t *testing.T) {
	setupNoConfig(t)

	_, _, err := executeCmd("restore", "passkey")
	if err == nil {
		t.Fatal("expected error for passkey restore in CLI")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAddressTransfer_NoTarget(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("address", "transfer", "alice", "--yes")
	if err == nil {
		t.Fatal("expected error without --to or --target-key")
	}
	if !strings.Contains(err.Error(), "--to") || !strings.Contains(err.Error(), "--target-key") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestWalletList_NoConfig(t *testing.T) {
	setupNoConfig(t)

	_, _, err := executeCmd("wallet", "list")
	if err == nil {
		t.Fatal("expected error without config")
	}
	if !strings.Contains(err.Error(), "no config found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKeyShow_NoConfig(t *testing.T) {
	setupNoConfig(t)

	_, _, err := executeCmd("key", "show")
	if err == nil {
		t.Fatal("expected error without config")
	}
	if !strings.Contains(err.Error(), "no config found") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInvoiceCreate_ZeroAmount(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("invoice", "create", "--amount", "0")
	if err == nil {
		t.Fatal("expected error for zero amount")
	}
	if !strings.Contains(err.Error(), "positive integer") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestKeyRotate_NonNumericSlot(t *testing.T) {
	setupConfig(t, testConfig())

	_, _, err := executeCmd("key", "rotate", "abc", "--yes")
	if err == nil {
		t.Fatal("expected error for non-numeric slot")
	}
	if !strings.Contains(err.Error(), "slot must be 0") {
		t.Errorf("unexpected error: %v", err)
	}
}
