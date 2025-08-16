#!/bin/sh
set -e

VERSION="v1.0.0"
BINARY="gessage"
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

if [ "$ARCH" = "x86_64" ]; then
    ARCH="amd64"
elif [ "$ARCH" = "arm64" ] || [ "$ARCH" = "aarch64" ]; then
    ARCH="arm64"
fi

URL="https://github.com/gessage/gessage-cli/releases/download/$VERSION/$BINARY-$OS-$ARCH"

echo "Downloading $URL..."
curl -L $URL -o /tmp/$BINARY
chmod +x /tmp/$BINARY
sudo mv /tmp/$BINARY /usr/local/bin/$BINARY

echo "$BINARY installed successfully!"
$BINARY --version