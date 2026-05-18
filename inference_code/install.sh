#!/bin/bash
set -e

echo "Installing system dependencies (libpcap)..."
if command -v apt-get >/dev/null; then
    sudo apt-get update
    sudo apt-get install -y libpcap-dev wget tar
elif command -v dnf >/dev/null; then
    sudo dnf install -y libpcap-devel wget tar
elif command -v yum >/dev/null; then
    sudo yum install -y libpcap-devel wget tar
else
    echo "Package manager was not detected. Install libpcap-dev/libpcap-devel manually."
fi

VERSION="1.24.2"
ARCH=$(uname -m)

if [ "$ARCH" = "x86_64" ]; then
    PKG="onnxruntime-linux-x64-$VERSION"
elif [ "$ARCH" = "aarch64" ]; then
    PKG="onnxruntime-linux-aarch64-$VERSION"
else
    echo "Error: architecture $ARCH is not supported by this script."
    exit 1
fi

URL="https://github.com/microsoft/onnxruntime/releases/download/v$VERSION/$PKG.tgz"

echo "Downloading ONNX Runtime v$VERSION..."
wget -q --show-progress "$URL" -O onnx.tgz

echo "Extracting..."
tar -xzf onnx.tgz

echo "Copying libraries to /usr/local/lib..."
sudo cp -P $PKG/lib/libonnxruntime.so* /usr/local/lib/

sudo ln -sf /usr/local/lib/libonnxruntime.so.$VERSION /usr/local/lib/onnxruntime.so

echo "Updating linker cache..."
sudo ldconfig

echo "Removing temporary files..."
rm -rf onnx.tgz $PKG

echo "Installation completed."
