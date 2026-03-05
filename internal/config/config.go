package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	lnbot "github.com/lnbotdev/go-sdk"
)

// Config stores the CLI authentication state.
// Only the user key and active wallet ID are persisted locally.
// Wallet listing comes from the API.
type Config struct {
	PrimaryKey     string `json:"primary_key"`
	SecondaryKey   string `json:"secondary_key,omitempty"`
	ActiveWalletID string `json:"active_wallet_id,omitempty"`
}

func Path() string {
	if p := os.Getenv("LNBOT_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "lnbot", "config.json")
}

func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if cfg.PrimaryKey == "" {
		return nil, nil
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	p := Path()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, append(data, '\n'), 0o600)
}

func Init(primaryKey, secondaryKey, walletID string) (*Config, error) {
	cfg := &Config{
		PrimaryKey:     primaryKey,
		SecondaryKey:   secondaryKey,
		ActiveWalletID: walletID,
	}
	return cfg, cfg.Save()
}

// Client returns an authenticated API client using the user key.
func (c *Config) Client() *lnbot.Client {
	return lnbot.New(c.PrimaryKey)
}

// AnonClient returns an unauthenticated API client.
func AnonClient() *lnbot.Client {
	return lnbot.New("")
}
