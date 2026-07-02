#!/usr/bin/env sh
set -eu

REPO="crevas/Apple-Ads-CLI"
BINARY="lily"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin) os="darwin" ;;
  linux) os="linux" ;;
  *)
    echo "Unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  arm64|aarch64) arch="arm64" ;;
  x86_64|amd64) arch="amd64" ;;
  *)
    echo "Unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

version="${LILY_ADS_CLI_VERSION:-latest}"
if [ "$version" = "latest" ]; then
  version="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1)"
fi

if [ -z "$version" ]; then
  echo "Could not resolve latest release for $REPO" >&2
  exit 1
fi

archive="${BINARY}_${os}_${arch}.tar.gz"
url="https://github.com/$REPO/releases/download/$version/$archive"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

echo "Downloading $url"
curl -fsSL "$url" -o "$tmp_dir/$archive"
tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"

mkdir -p "$INSTALL_DIR"
if [ -w "$INSTALL_DIR" ]; then
  mv "$tmp_dir/$BINARY" "$INSTALL_DIR/$BINARY"
else
  sudo mv "$tmp_dir/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "Installed $BINARY to $INSTALL_DIR/$BINARY"
