#!/usr/bin/env sh
set -eu

OWNER="kuopenx"
REPO="figma-asset"
LOCAL_BIN="$HOME/.local/bin"
PLUGIN_DIR="$HOME/figma-asset-plugin"

# Detect platform.
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

ZIP_NAME="figma-asset-$OS-$ARCH.zip"
DOWNLOAD_URL="https://github.com/$OWNER/$REPO/releases/latest/download/$ZIP_NAME"
CHECKSUMS_URL="https://github.com/$OWNER/$REPO/releases/latest/download/checksums.txt"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Detected platform: $OS/$ARCH"
echo "Downloading $ZIP_NAME..."

# Download zip.
ZIP_PATH="$TMP_DIR/$ZIP_NAME"
curl -fsSL -o "$ZIP_PATH" "$DOWNLOAD_URL"

# Download checksums and verify.
CHECKSUMS_PATH="$TMP_DIR/checksums.txt"
curl -fsSL -o "$CHECKSUMS_PATH" "$CHECKSUMS_URL"

EXPECTED=$(grep "$ZIP_NAME" "$CHECKSUMS_PATH" | awk '{print $1}')
if [ -z "$EXPECTED" ]; then
    echo "No checksum found for $ZIP_NAME" >&2
    exit 1
fi

ACTUAL=$(sha256sum "$ZIP_PATH" | awk '{print $1}')
if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Checksum mismatch: expected $EXPECTED, got $ACTUAL" >&2
    exit 1
fi
echo "Checksum verified."

# Extract.
unzip -o "$ZIP_PATH" -d "$TMP_DIR/extract"

# Install binary to ~/.local/bin.
mkdir -p "$LOCAL_BIN"
cp -f "$TMP_DIR/extract/figma-asset" "$LOCAL_BIN/figma-asset"
chmod +x "$LOCAL_BIN/figma-asset"

# Install plugin files to ~/figma-asset-plugin.
mkdir -p "$PLUGIN_DIR"
cp -f "$TMP_DIR/extract/manifest.json" "$PLUGIN_DIR/manifest.json"
if [ -d "$TMP_DIR/extract/plugin" ]; then
    mkdir -p "$PLUGIN_DIR/plugin"
    cp -f "$TMP_DIR/extract/plugin/"* "$PLUGIN_DIR/plugin/"
fi

echo
echo "========================================="
echo "  Installation complete!"
echo "========================================="
echo
echo "Binary location:"
echo "  $LOCAL_BIN/figma-asset"
echo
echo "Plugin location:"
echo "  $PLUGIN_DIR/manifest.json"
echo "  $PLUGIN_DIR/plugin/code.js"
echo "  $PLUGIN_DIR/plugin/ui.html"
echo

# PATH check.
case ":$PATH:" in
    *":$LOCAL_BIN:"*) ;;
    *)
        echo "-----------------------------------------"
        echo "  ACTION REQUIRED: Add to PATH"
        echo "-----------------------------------------"
        echo "  $LOCAL_BIN is not in your PATH."
        echo "  Add this to ~/.zshrc or ~/.bashrc:"
        echo
        echo "    export PATH=\"\$HOME/.local/bin:\$PATH\""
        echo
        echo "  Then restart your terminal."
        echo
        ;;
esac

echo "-----------------------------------------"
echo "  Import plugin into Figma (one-time)"
echo "-----------------------------------------"
echo "  1. Open Figma Desktop"
echo "  2. Menu: Plugins -> Development -> Import plugin from manifest..."
echo "  3. Select: $PLUGIN_DIR/manifest.json"
echo
echo "  After import, run:"
echo "    figma-asset start"
echo "    figma-asset export png --platform flutter --node <node-id> --out ./assets"
echo
echo "  To check for updates:"
echo "    figma-asset upgrade --check"
echo "  To update:"
echo "    figma-asset upgrade"