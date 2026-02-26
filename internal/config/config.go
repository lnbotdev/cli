package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	lnbot "github.com/lnbotdev/go-sdk"
)

type WalletEntry struct {
	ID           string `json:"id"`
	PrimaryKey   string `json:"primary_key"`
	SecondaryKey string `json:"secondary_key"`
	Address      string `json:"address"`
}

type Config struct {
	Active  string                 `json:"active"`
	Wallets map[string]WalletEntry `json:"wallets"`
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

func Init() (*Config, error) {
	cfg := &Config{Wallets: make(map[string]WalletEntry)}
	return cfg, cfg.Save()
}

func (c *Config) ActiveWallet() (*WalletEntry, string, error) {
	if c == nil || len(c.Wallets) == 0 {
		return nil, "", fmt.Errorf("no wallets configured — run 'lnbot wallet create --name <n>'")
	}
	if c.Active == "" {
		return nil, "", fmt.Errorf("no active wallet — run 'lnbot wallet use <name>'")
	}
	w, ok := c.Wallets[c.Active]
	if !ok {
		return nil, "", fmt.Errorf("active wallet %q not found in config", c.Active)
	}
	return &w, c.Active, nil
}

func (c *Config) ResolveWallet(name string) (*WalletEntry, string, error) {
	if name == "" {
		return c.ActiveWallet()
	}
	w, ok := c.Wallets[name]
	if !ok {
		return nil, "", fmt.Errorf("wallet %q not found in config", name)
	}
	return &w, name, nil
}

func (c *Config) Client(walletName string) (*lnbot.Client, *WalletEntry, string, error) {
	w, name, err := c.ResolveWallet(walletName)
	if err != nil {
		return nil, nil, "", err
	}
	return lnbot.New(w.PrimaryKey), w, name, nil
}

func AnonClient() *lnbot.Client {
	return lnbot.New("")
}
