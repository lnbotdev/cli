# lnbot CLI

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
# Create config + first wallet
lnbot init
lnbot wallet create --name agent01

# Receive sats (prints QR, waits for payment)
lnbot invoice create --amount 1000 --memo "first payment"

# Send sats
lnbot pay alice@ln.bot --amount 500

# Check balance
lnbot balance
```

## Commands

```
Getting Started:
  init              Create local config file
  wallet            Create, list, switch, rename, delete wallets

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
  restore           Restore wallet from passphrase or passkey

Integrations:
  webhook           Register, list, delete webhook endpoints
  mcp               MCP server config for AI agents
```

Every command supports `--help` for detailed usage, flags, and examples.

## Global flags

| Flag | Description |
|---|---|
| `-w, --wallet <name>` | Target a specific wallet |
| `--json` | Output as JSON (machine-readable) |
| `-y, --yes` | Skip confirmation prompts |

## Configuration

Config is stored at `~/.config/lnbot/config.json`. Override the path with `LNBOT_CONFIG` env var.

## MCP integration

Generate config for AI agents (Claude, Cursor, etc.):

```bash
lnbot mcp config --remote
```

## Shell completions

```bash
source <(lnbot completion bash)   # Bash
source <(lnbot completion zsh)    # Zsh
lnbot completion fish | source    # Fish
```

## License

MIT
