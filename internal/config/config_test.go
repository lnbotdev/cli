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
	data := `{"primary_key":"uk_abc123","secondary_key":"uk_def456","active_wallet_id":"wal_xyz"}`
	os.WriteFile(p, []byte(data), 0o600)
	t.Setenv("LNBOT_CONFIG", p)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.PrimaryKey != "uk_abc123" {
		t.Errorf("PrimaryKey = %q, want %q", cfg.PrimaryKey, "uk_abc123")
	}
	if cfg.SecondaryKey != "uk_def456" {
		t.Errorf("SecondaryKey = %q, want %q", cfg.SecondaryKey, "uk_def456")
	}
	if cfg.ActiveWalletID != "wal_xyz" {
		t.Errorf("ActiveWalletID = %q, want %q", cfg.ActiveWalletID, "wal_xyz")
	}
}

func TestLoad_EmptyPrimaryKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	data := `{"primary_key":"","active_wallet_id":"wal_xyz"}`
	os.WriteFile(p, []byte(data), 0o600)
	t.Setenv("LNBOT_CONFIG", p)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg != nil {
		t.Errorf("Load() should return nil for empty primary key, got %+v", cfg)
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

	cfg, err := Init("uk_test", "uk_sec", "wal_123")
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if cfg.PrimaryKey != "uk_test" {
		t.Errorf("PrimaryKey = %q, want %q", cfg.PrimaryKey, "uk_test")
	}
	if cfg.SecondaryKey != "uk_sec" {
		t.Errorf("SecondaryKey = %q, want %q", cfg.SecondaryKey, "uk_sec")
	}
	if cfg.ActiveWalletID != "wal_123" {
		t.Errorf("ActiveWalletID = %q, want %q", cfg.ActiveWalletID, "wal_123")
	}

	if _, err := os.Stat(p); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestSaveAndLoad_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	t.Setenv("LNBOT_CONFIG", p)

	original := &Config{
		PrimaryKey:     "uk_primary",
		SecondaryKey:   "uk_secondary",
		ActiveWalletID: "wal_abc",
	}
	if err := original.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

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

	cfg := &Config{PrimaryKey: "uk_test"}
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

func TestClient_ReturnsClient(t *testing.T) {
	cfg := &Config{
		PrimaryKey:     "uk_test123",
		ActiveWalletID: "wal_abc",
	}
	client := cfg.Client()
	if client == nil {
		t.Fatal("Client() returned nil")
	}
}

func TestAnonClient(t *testing.T) {
	client := AnonClient()
	if client == nil {
		t.Fatal("AnonClient() returned nil")
	}
}

func TestLoad_OldConfigFormat(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	// Old format with wallets map — should be treated as no config (no primary_key)
	data := `{"active":"w1","wallets":{"w1":{"id":"wal_123","primary_key":"pk"}}}`
	os.WriteFile(p, []byte(data), 0o600)
	t.Setenv("LNBOT_CONFIG", p)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg != nil {
		t.Errorf("Load() should return nil for old config format, got %+v", cfg)
	}
}
