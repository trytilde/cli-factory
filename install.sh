#!/usr/bin/env bash
set -euo pipefail

REPO="trytilde/cli-factory"
INSTALL_DIR="${FACTORY_INSTALL_DIR:-$HOME/.factory/bin}"
VERSION="${FACTORY_VERSION:-latest}"

usage() {
  cat <<'EOF'
Install factory from GitHub Releases.

Usage:
  install.sh [--version vX.Y.Z] [--install-dir PATH]

Environment:
  FACTORY_VERSION       Release tag to install. Defaults to latest.
  FACTORY_INSTALL_DIR   Install directory. Defaults to ~/.factory/bin.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$(uname -s)" in
  Linux) os="linux" ;;
  Darwin) os="darwin" ;;
  MINGW*|MSYS*|CYGWIN*) os="windows" ;;
  *) echo "Unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

if [[ "$os" == "windows" && "$arch" != "amd64" ]]; then
  echo "Windows arm64 release artifact is not published yet." >&2
  exit 1
fi

artifact="factory-${os}-${arch}"
binary="factory"
if [[ "$os" == "windows" ]]; then
  artifact="${artifact}.exe"
  binary="factory.exe"
fi

if [[ "$VERSION" == "latest" ]]; then
  url="https://github.com/${REPO}/releases/latest/download/${artifact}.tar.gz"
else
  url="https://github.com/${REPO}/releases/download/${VERSION}/${artifact}.tar.gz"
fi

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

echo "Downloading ${url}"
curl -fsSL "$url" -o "$tmp/factory.tar.gz"
tar -xzf "$tmp/factory.tar.gz" -C "$tmp"

mkdir -p "$INSTALL_DIR"
install -m 0755 "$tmp/$artifact" "$INSTALL_DIR/$binary"

echo "Installed $binary to $INSTALL_DIR/$binary"
case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "Add to PATH: export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
esac

