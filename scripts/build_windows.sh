#!/usr/bin/env bash
set -e

echo "Building Windows portable release..."

APP_NAME="KworkAssistant-Windows-x64"
DIST_DIR="dist/$APP_NAME"

# Clean up
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
mkdir -p "$DIST_DIR/data"
mkdir -p "$DIST_DIR/logs"

# Build Windows EXE
echo "Compiling for Windows/amd64..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o "$DIST_DIR/kwork-assistant.exe" ./cmd/app

# Copy configuration and scripts
echo "Copying resources..."
cp .env.example "$DIST_DIR/.env.example"
cp scripts/windows/*.ps1 "$DIST_DIR/"
cp docs/README_WINDOWS_DRAFT.md "$DIST_DIR/README_WINDOWS.md"

# Package Zip
echo "Zipping archive..."
cd "$DIST_DIR"
rm -f "../$APP_NAME.zip"
zip -r "../$APP_NAME.zip" .
cd ../..

echo "Done! Windows release archive created at dist/$APP_NAME.zip"
