#!/usr/bin/env bash
#
# One-command installer for grok-proxy-cli.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/deivid22srk/<repo>/main/install.sh | bash
#
# Or, after clone:
#   ./install.sh
#
# This script:
#   1. Detects the OS and architecture
#   2. If a prebuilt binary exists in the latest GitHub Release, downloads it
#   3. Otherwise, bootstraps Go (if missing), builds from source, and installs
#   4. Installs the binary as `grok-proxy-cli` to ~/.local/bin (or /usr/local/bin if root)
#   5. Prints next-step instructions
#
# The desktop (Wails) app is intentionally NOT built — this is the terminal edition.

set -euo pipefail

REPO_SLUG="${GROK_PROXY_REPO:-deivid22srk/grok-proxy-cli}"
BIN_NAME="grok-proxy-cli"

# ---------- helpers ----------
err()  { printf "\033[31merror:\033[0m %s\n" "$*" >&2; }
info() { printf "\033[36m%s\033[0m\n" "$*"; }
ok()   { printf "\033[32m✓\033[0m %s\n" "$*"; }

# ---------- detect platform ----------
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$OS" in
  linux)  OS="linux"  ;;
  darwin) OS="darwin" ;;
  *)      err "unsupported OS: $OS"; exit 1 ;;
esac
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) err "unsupported arch: $ARCH"; exit 1 ;;
esac

info "platform: ${OS}/${ARCH}"

# ---------- pick install dir ----------
INSTALL_DIR="${INSTALL_DIR:-}"
if [ -z "$INSTALL_DIR" ]; then
  if [ "$(id -u)" = "0" ]; then
    INSTALL_DIR="/usr/local/bin"
  else
    INSTALL_DIR="$HOME/.local/bin"
  fi
fi
mkdir -p "$INSTALL_DIR"

# ---------- try prebuilt binary from latest release ----------
try_download_binary() {
  local url
  url=$(curl -fsSL "https://api.github.com/repos/${REPO_SLUG}/releases/latest" 2>/dev/null \
        | grep -oE "https://[^\"']*${OS}-${ARCH}[^\"']*" | head -n1 || true)
  if [ -z "$url" ]; then
    return 1
  fi
  info "downloading prebuilt binary: $url"
  local tmp="/tmp/${BIN_NAME}.download"
  if ! curl -fSL -o "$tmp" "$url"; then
    return 1
  fi
  chmod +x "$tmp"
  mv "$tmp" "${INSTALL_DIR}/${BIN_NAME}"
  ok "installed ${BIN_NAME} to ${INSTALL_DIR}"
  return 0
}

# ---------- bootstrap Go ----------
ensure_go() {
  if command -v go >/dev/null 2>&1; then
    return 0
  fi
  local version="1.23.4"
  local tarball="go${version}.${OS}-${ARCH}.tar.gz"
  local url="https://go.dev/dl/${tarball}"
  local tmpdir="${TMPDIR:-/tmp}/go-install-$$"
  mkdir -p "$tmpdir"
  info "downloading Go ${version} — ${url}"
  if ! curl -fSL -o "${tmpdir}/${tarball}" "$url"; then
    err "failed to download Go"
    rm -rf "$tmpdir"
    return 1
  fi
  info "extracting Go to ${tmpdir}"
  tar -C "$tmpdir" -xzf "${tmpdir}/${tarball}"
  export PATH="${tmpdir}/go/bin:${PATH}"
  ok "go $(go version) ready (temporary)"
  # leave a hint for the caller to make permanent
  echo "${tmpdir}/go/bin" > /tmp/grok-proxy-go-bin-path
  return 0
}

# ---------- build from source ----------
build_from_source() {
  local workdir
  workdir="$(mktemp -d)"
  info "cloning ${REPO_SLUG} to ${workdir}"
  if ! git clone --depth 1 "https://github.com/${REPO_SLUG}.git" "$workdir/repo"; then
    err "git clone failed"
    return 1
  fi
  if ! ensure_go; then
    return 1
  fi
  info "building ${BIN_NAME} (this can take a minute on first run)"
  (
    cd "$workdir/repo"
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
      -o "${INSTALL_DIR}/${BIN_NAME}" ./cmd/grok-proxy-cli
  )
  chmod +x "${INSTALL_DIR}/${BIN_NAME}"
  rm -rf "$workdir"
  ok "installed ${BIN_NAME} to ${INSTALL_DIR}"
  return 0
}

# ---------- main ----------
if ! try_download_binary; then
  info "no prebuilt binary available — building from source"
  if ! build_from_source; then
    err "install failed"
    exit 1
  fi
fi

# ---------- path hint ----------
case ":${PATH}:" in
  *":${INSTALL_DIR}:"*) ;;
  *)
    echo
    info "note: ${INSTALL_DIR} is not in your PATH"
    info "add this to your shell rc (~/.bashrc, ~/.zshrc, …):"
    echo "    export PATH=\"${INSTALL_DIR}:\$PATH\""
    ;;
esac

# ---------- go hint ----------
if [ -f /tmp/grok-proxy-go-bin-path ]; then
  GOBIN=$(cat /tmp/grok-proxy-go-bin-path)
  echo
  info "Go was bootstrapped to a temporary directory."
  info "To make it permanent, add to your shell rc:"
  echo "    export PATH=\"${GOBIN}:\$PATH\""
  rm -f /tmp/grok-proxy-go-bin-path
fi

echo
ok "done. Next steps:"
echo "    ${BIN_NAME} login      # sign in to xAI"
echo "    ${BIN_NAME} accounts   # check your account"
echo "    ${BIN_NAME} serve      # start the local proxy at http://127.0.0.1:8787/v1"
echo "    ${BIN_NAME} chat       # interactive REPL"
echo "    ${BIN_NAME} ask \"hi\"   # one-shot prompt"
