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

# Receive ₿1,000 (prints BOLT11, waits for payment)
lnbot invoice create --amount 1000 --memo "first payment"

# Send ₿500 (shows the fees and the total, then asks)
lnbot pay alice@ln.bot --amount 500

# Check balance
lnbot balance
```

## Commands

```
Getting started:
  init              Register account and create first wallet
  wallet            Create, list, switch, and rename wallets

Money:
  balance           Show wallet balance
  invoice           Create and list Lightning invoices
  pay               Send bitcoin to an address or invoice
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
| `--json` | Print JSON instead of text (machine-readable) |
| `-y, --yes` | Skip confirmation prompts |

## Amounts and fees

Amounts are whole sats, written the way [BIP 177](https://github.com/bitcoin/bips/blob/master/bip-0177.mediawiki) does: `--amount 1000` is ₿1,000.

Payments between ln.bot wallets are free. A payment out to Lightning pays a 0.25% service fee (at least ₿1) and the network fee, 0.25% plus Lightning routing. `lnbot pay` shows both and the total before it asks:

```
$ lnbot pay alice@example.com --amount 250000
  Amount:       ₿250,000
  Network fee:  ₿700 (estimate)
  Service fee:  ₿625
  Total:        ₿251,325
Send ₿250,000 to alice@example.com? (y/N) y
✓ Payment settled in 1840ms
  Amount:     ₿250,000
  Fees:       ₿1,323 (network ₿698, service ₿625)
  Total:      ₿251,323
  Available:  ₿48,675
```

The network fee is capped at 1% of the amount (at least ₿20). Pass `--max-fee <sats>` to allow more, and `--all` instead of `--amount` to send everything the wallet can to a Lightning address or LNURL. `--yes` skips the quote along with the question.

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
