package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPath_Default(t *testing.T) {
	t.Setenv("LNBOT_CONFIG", "")
	p := Path()
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "lnbot", "config.json")
	if p != want {
		t.Errorf("Path() = %q, want %q", p, want)
	}
}

func TestPath_EnvOverride(t *testing.T) {
	t.Setenv("LNBOT_CONFIG", "/tmp/custom/config.json")
	if got := Path(); got != "/tmp/custom/config.json" {
		t.Errorf("Path() = %q, want /tmp/custom/config.json", got)
	}
}

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LNBOT_CONFIG", filepath.Join(dir, "nonexistent.json"))
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg != nil {
		t.Errorf("Load() = %+v, want nil", cfg)
	}
}

func TestLoad_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	data := `{"active":"w1","wallets":{"w1":{"id":"wal_123","primary_key":"pk","secondary_key":"sk","address":"a@ln.bot"}}}`
	os.WriteFile(p, []byte(data), 0o600)
	t.Setenv("LNBOT_CONFIG", p)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Active != "w1" {
		t.Errorf("Active = %q, want %q", cfg.Active, "w1")
	}
	if len(cfg.Wallets) != 1 {
		t.Errorf("len(Wallets) = %d, want 1", len(cfg.Wallets))
	}
	w := cfg.Wallets["w1"]
	if w.ID != "wal_123" {
		t.Errorf("ID = %q, want %q", w.ID, "wal_123")
	}
	if w.PrimaryKey != "pk" {
		t.Errorf("PrimaryKey = %q, want %q", w.PrimaryKey, "pk")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	os.WriteFile(p, []byte("{bad json"), 0o600)
	t.Setenv("LNBOT_CONFIG", p)

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid JSON")
	}
}

func TestInit_CreatesConfig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "config.json")
	t.Setenv("LNBOT_CONFIG", p)

	cfg, err := Init()
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg == nil {
		t.Fatal("Init() returned nil")
	}
	if cfg.Wallets == nil {
		t.Error("Init() Wallets map is nil")
	}

	// Verify file exists
	if _, err := os.Stat(p); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSaveAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", p)

	original := &Config{
		Active: "prod",
		Wallets: map[string]WalletEntry{
			"prod": {
				ID:           "wal_abc",
				PrimaryKey:   "key_primary",
				SecondaryKey: "key_secondary",
				Address:      "prod@ln.bot",
			},
			"staging": {
				ID:         "wal_def",
				PrimaryKey: "key_staging",
			},
		},
	}
	if err := original.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Compare via JSON for deep equality
	origJSON, _ := json.Marshal(original)
	loadedJSON, _ := json.Marshal(loaded)
	if string(origJSON) != string(loadedJSON) {
		t.Errorf("round-trip mismatch:\n  saved:  %s\n  loaded: %s", origJSON, loadedJSON)
	}
}

func TestSave_FilePermissions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", p)

	cfg := &Config{Wallets: make(map[string]WalletEntry)}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	info, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file permissions = %o, want 600", perm)
	}
}

func TestActiveWallet_NoWallets(t *testing.T) {
	cfg := &Config{Wallets: map[string]WalletEntry{}}
	_, _, err := cfg.ActiveWallet()
	if err == nil {
		t.Fatal("expected error for no wallets")
	}
}

func TestActiveWallet_NilConfig(t *testing.T) {
	var cfg *Config
	_, _, err := cfg.ActiveWallet()
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestActiveWallet_NoActiveSet(t *testing.T) {
	cfg := &Config{
		Wallets: map[string]WalletEntry{"w1": {ID: "wal_1"}},
	}
	_, _, err := cfg.ActiveWallet()
	if err == nil {
		t.Fatal("expected error for no active wallet")
	}
}

func TestActiveWallet_ActiveNotFound(t *testing.T) {
	cfg := &Config{
		Active:  "missing",
		Wallets: map[string]WalletEntry{"w1": {ID: "wal_1"}},
	}
	_, _, err := cfg.ActiveWallet()
	if err == nil {
		t.Fatal("expected error for active wallet not found")
	}
}

func TestActiveWallet_Success(t *testing.T) {
	cfg := &Config{
		Active: "w1",
		Wallets: map[string]WalletEntry{
			"w1": {ID: "wal_1", PrimaryKey: "pk"},
		},
	}
	entry, name, err := cfg.ActiveWallet()
	if err != nil {
		t.Fatalf("ActiveWallet() error = %v", err)
	}
	if name != "w1" {
		t.Errorf("name = %q, want %q", name, "w1")
	}
	if entry.ID != "wal_1" {
		t.Errorf("ID = %q, want %q", entry.ID, "wal_1")
	}
}

func TestResolveWallet_EmptyName(t *testing.T) {
	cfg := &Config{
		Active: "w1",
		Wallets: map[string]WalletEntry{
			"w1": {ID: "wal_1"},
		},
	}
	entry, name, err := cfg.ResolveWallet("")
	if err != nil {
		t.Fatalf("ResolveWallet() error = %v", err)
	}
	if name != "w1" {
		t.Errorf("name = %q, want %q", name, "w1")
	}
	if entry.ID != "wal_1" {
		t.Errorf("ID = %q, want %q", entry.ID, "wal_1")
	}
}

func TestResolveWallet_SpecificName(t *testing.T) {
	cfg := &Config{
		Active: "w1",
		Wallets: map[string]WalletEntry{
			"w1": {ID: "wal_1"},
			"w2": {ID: "wal_2"},
		},
	}
	entry, name, err := cfg.ResolveWallet("w2")
	if err != nil {
		t.Fatalf("ResolveWallet() error = %v", err)
	}
	if name != "w2" {
		t.Errorf("name = %q, want %q", name, "w2")
	}
	if entry.ID != "wal_2" {
		t.Errorf("ID = %q, want %q", entry.ID, "wal_2")
	}
}

func TestResolveWallet_NotFound(t *testing.T) {
	cfg := &Config{
		Active:  "w1",
		Wallets: map[string]WalletEntry{"w1": {ID: "wal_1"}},
	}
	_, _, err := cfg.ResolveWallet("missing")
	if err == nil {
		t.Fatal("expected error for wallet not found")
	}
}

func TestClient_ReturnsClient(t *testing.T) {
	cfg := &Config{
		Active: "w1",
		Wallets: map[string]WalletEntry{
			"w1": {ID: "wal_1", PrimaryKey: "pk_123"},
		},
	}
	client, entry, name, err := cfg.Client("")
	if err != nil {
		t.Fatalf("Client() error = %v", err)
	}
	if client == nil {
		t.Fatal("Client() returned nil client")
	}
	if name != "w1" {
		t.Errorf("name = %q, want %q", name, "w1")
	}
	if entry.PrimaryKey != "pk_123" {
		t.Errorf("PrimaryKey = %q, want %q", entry.PrimaryKey, "pk_123")
	}
}

func TestAnonClient(t *testing.T) {
	client := AnonClient()
	if client == nil {
		t.Fatal("AnonClient() returned nil")
	}
}
