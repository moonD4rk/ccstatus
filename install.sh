#!/usr/bin/env bash
# Install ccstatus: detect OS/arch, download the latest release, verify, install, clean up.
set -euo pipefail

REPO="chaudhryfaisal/ccstatus"
BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"
VERSION="latest"
CONFIGURE=0

usage() {
  cat <<USAGE
Usage: install.sh [--configure] [--version vX.Y.Z] [--bin-dir DIR]

  --configure       write default settings, register with Claude Code, run preview
  --version TAG     install a specific release instead of the latest
  --bin-dir DIR     install directory (default: \$HOME/.local/bin)
USAGE
}

while [ $# -gt 0 ]; do
  case "$1" in
    --configure) CONFIGURE=1 ;;
    --version) VERSION="${2:?--version needs a value}"; shift ;;
    --bin-dir) BIN_DIR="${2:?--bin-dir needs a value}"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
  esac
  shift
done

case "$(uname -s)" in
  Darwin) OS=darwin ;;
  Linux)  OS=linux ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64|amd64)  ARCH=x86_64 ;;
  arm64|aarch64) ARCH=arm64 ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

ASSET="ccstatus_${OS}_${ARCH}.tar.gz"
if [ "$VERSION" = "latest" ]; then
  BASE_URL="https://github.com/${REPO}/releases/latest/download"
else
  BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
fi

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

echo "Downloading ${ASSET} (${VERSION})..."
curl -fsSL -o "$TMP/$ASSET" "$BASE_URL/$ASSET"
curl -fsSL -o "$TMP/checksums.txt" "$BASE_URL/checksums.txt"

EXPECTED="$(awk -v f="$ASSET" '$2 == f {print $1}' "$TMP/checksums.txt")"
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "$TMP/$ASSET" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "$TMP/$ASSET" | awk '{print $1}')"
fi
if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
  echo "Checksum mismatch for ${ASSET}" >&2
  exit 1
fi

tar -xzf "$TMP/$ASSET" -C "$TMP" ccstatus
mkdir -p "$BIN_DIR"
install -m 0755 "$TMP/ccstatus" "$BIN_DIR/ccstatus"
echo "Installed $("$BIN_DIR/ccstatus" --version) to $BIN_DIR/ccstatus"

if [ "$CONFIGURE" = 1 ]; then
  CFG="${XDG_CONFIG_HOME:-$HOME/.config}/ccstatus/settings.json"
  [ -f "$CFG" ] || "$BIN_DIR/ccstatus" init
  # install registers the command as "ccstatus", so it must be on PATH to run
  "$BIN_DIR/ccstatus" install
  "$BIN_DIR/ccstatus" validate
  "$BIN_DIR/ccstatus" preview
  echo "Configured. Restart Claude Code to see the status line."
fi

case ":$PATH:" in
  *":$BIN_DIR:"*) ;;
  *) echo "Note: add $BIN_DIR to your PATH, e.g. export PATH=\"$BIN_DIR:\$PATH\"" ;;
esac
