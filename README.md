# ln.bot CLI

Command-line interface for [ln.bot](https://ln.bot) — Bitcoin Lightning wallets for AI agents.

## Install

### One-liner (macOS, Linux)

```bash
curl -fsSL https://ln.bot/install.sh | bash
```

### Homebrew

```bash
brew install lnbotdev/tap/lnbot
```

### Go

```bash
go install github.com/lnbotdev/cli@latest
```

### Binary download

Grab the latest release for your platform from the [releases page](https://github.com/lnbotdev/cli/releases).

## Quick start

```bash
# Register account + create first wallet
lnbot init

# Create another wallet
lnbot wallet create

# Receive sats (prints BOLT11, waits for payment)
lnbot invoice create --amount 1000 --memo "first payment"

# Send sats
lnbot pay alice@ln.bot --amount 500

# Check balance
lnbot balance
```

## Commands

```
Getting Started:
  init              Register account and create first wallet
  wallet            Create, list, switch, and rename wallets

Money:
  balance           Show wallet balance
  invoice           Create and list Lightning invoices
  pay               Send sats to an address or invoice
  payment           List outgoing payments
  transactions      List all transaction history

Identity:
  address           Manage Lightning addresses (buy, list, transfer, delete)
  whoami            Show current wallet info
  status            Wallet status and API health

Security:
  key               Show or rotate API keys
  backup            Generate recovery passphrase or register passkey
  restore           Restore account from passphrase or passkey

Integrations:
  webhook           Register, list, delete webhook endpoints
  mcp               MCP server config for AI agents
```

Every command supports `--help` for detailed usage, flags, and examples.

## Global flags

| Flag | Description |
|---|---|
| `-w, --wallet <id\|name>` | Target a specific wallet (ID or name) |
| `--json` | Output as JSON (machine-readable) |
| `-y, --yes` | Skip confirmation prompts |

## Multi-wallet

All wallets share a single user key (`uk_`). The CLI stores only the user key and the active wallet ID locally — wallet data comes from the API.

```bash
# Create wallets
lnbot wallet create                  # auto-named
lnbot wallet rename production       # rename it

# Switch active wallet
lnbot wallet use agent01             # by name
lnbot wallet use wal_7x9kQ2mR       # by ID

# Target a specific wallet for one command
lnbot balance --wallet wal_abc
lnbot pay alice@ln.bot --amount 100 --wallet agent01
```

## Configuration

Config is stored at `~/.config/lnbot/config.json`. Override the path with `LNBOT_CONFIG` env var.

```json
{
  "primary_key": "uk_...",
  "secondary_key": "uk_...",
  "active_wallet_id": "wal_..."
}
```

## MCP integration

Generate config for AI agents (Claude, Cursor, etc.):

```bash
lnbot mcp config --remote
lnbot mcp config --remote --wallet wal_abc
```

## Shell completions

```bash
source <(lnbot completion bash)   # Bash
source <(lnbot completion zsh)    # Zsh
lnbot completion fish | source    # Fish
```

## License

MIT
