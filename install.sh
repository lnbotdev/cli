#!/bin/sh
# ln.bot CLI installer
# Usage: curl -fsSL https://ln.bot/install.sh | bash
#
# Detects OS and architecture, downloads the latest release from GitHub,
# and installs the binary to /usr/local/bin (or ~/.local/bin as fallback).

set -e

REPO="lnbotdev/cli"
BINARY="lnbot"
INSTALL_DIR="/usr/local/bin"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

info()  { printf "\033[0;32m%s\033[0m\n" "$1"; }
warn()  { printf "\033[0;33m%s\033[0m\n" "$1"; }
error() { printf "\033[0;31merror: %s\033[0m\n" "$1" >&2; exit 1; }

need() {
  command -v "$1" > /dev/null 2>&1 || error "requires $1 — install it and retry"
}

# ---------------------------------------------------------------------------
# Detect platform
# ---------------------------------------------------------------------------

detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux"  ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) error "unsupported OS: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) error "unsupported architecture: $(uname -m)" ;;
  esac
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
  need curl
  need tar

  OS=$(detect_os)
  ARCH=$(detect_arch)

  info "Detecting platform... ${OS}/${ARCH}"

  # Get latest release tag
  LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')

  if [ -z "$LATEST" ]; then
    error "could not determine latest release from GitHub"
  fi

  info "Latest version: ${LATEST}"

  # Build download URL
  VERSION="${LATEST#v}"
  if [ "$OS" = "windows" ]; then
    ARCHIVE="${BINARY}_${OS}_${ARCH}.zip"
  else
    ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
  fi
  URL="https://github.com/${REPO}/releases/download/${LATEST}/${ARCHIVE}"

  # Download
  TMPDIR=$(mktemp -d)
  trap 'rm -rf "$TMPDIR"' EXIT

  info "Downloading ${URL}..."
  curl -fsSL -o "${TMPDIR}/${ARCHIVE}" "$URL" || error "download failed — check https://github.com/${REPO}/releases"

  # Extract
  info "Extracting..."
  if [ "$OS" = "windows" ]; then
    need unzip
    unzip -q "${TMPDIR}/${ARCHIVE}" -d "$TMPDIR"
  else
    tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"
  fi

  # Install
  if [ -w "$INSTALL_DIR" ]; then
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  elif command -v sudo > /dev/null 2>&1; then
    info "Installing to ${INSTALL_DIR} (requires sudo)..."
    sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
    mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
    warn "Installed to ${INSTALL_DIR} — make sure it's in your PATH"
  fi

  chmod +x "${INSTALL_DIR}/${BINARY}"

  info ""
  info "✓ lnbot ${VERSION} installed to ${INSTALL_DIR}/${BINARY}"
  info ""
  info "  Get started:"
  info "    lnbot init"
  info "    lnbot wallet create --name agent01"
  info ""
  info "  Docs: https://ln.bot/docs"
}

main
