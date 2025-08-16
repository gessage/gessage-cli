#!/usr/bin/env bash
set -euo pipefail

# Build matrix → bin/gessage-<os>-<arch>[.exe]
# Includes: darwin (arm64, amd64), linux (arm64, amd64), windows (amd64)

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$ROOT_DIR/bin"
PKG="./cmd/gessage"
VERSION=${VERSION:-$(git describe --tags --always 2>/dev/null || echo dev)}

mkdir -p "$BIN_DIR"

# Default to static builds
export CGO_ENABLED=${CGO_ENABLED:-0}

build_one() {
  local os="$1" arch="$2" ext=""
  if [[ "$os" == "windows" ]]; then
    ext=".exe"
  fi
  local out="${BIN_DIR}/gessage-${os}-${arch}${ext}"
  echo "==> Building ${out}"
  GOOS="$os" GOARCH="$arch" \
    go build -trimpath -ldflags "-s -w -X github.com/ispooya/gessage-cli/internal/cli.Version=${VERSION}" -o "$out" "$PKG"
}

build_one darwin  arm64
build_one darwin  amd64
build_one linux   arm64
build_one linux   amd64
build_one windows amd64

echo "\nAll binaries written to: $BIN_DIR (version: ${VERSION})"

