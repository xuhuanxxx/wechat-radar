#!/bin/bash
set -e

# Build Lark Radar macOS .app bundle
# Architecture: Swift menu bar + Go data service (no Web frontend)
# Usage: ./scripts/build-macos-app.sh
# Output: dist/Lark Radar.app

APP_NAME="Lark Radar"
BUNDLE_ID="com.lark-radar.app"
VERSION="0.1.0"

# Detect architecture
ARCH=$(uname -m)

# Paths
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
APP_DIR="$DIST_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

# Go binary name
GO_BINARY="lark-radar-server"
GO_BINARY_PATH="$PROJECT_ROOT/apps/data-service/$GO_BINARY"

echo "=== Building $APP_NAME for $ARCH ==="

# Clean and create directories
rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

# 1. Build Go data service
echo "[1/4] Building Go data service..."
cd "$PROJECT_ROOT/apps/data-service"

# Clean previous build
rm -f "$GO_BINARY"

# Build with optimizations
CGO_ENABLED=1 go build -ldflags="-s -w" -o "$GO_BINARY" main.go

if [ ! -f "$GO_BINARY_PATH" ]; then
  echo "ERROR: Go binary not found after build"
  exit 1
fi

echo "    Go binary built: $(ls -lh "$GO_BINARY_PATH" | awk '{print $5}')"

# 2. Copy Go binary to Resources
echo "[2/4] Copying Go binary to app bundle..."
cp "$GO_BINARY_PATH" "$RESOURCES_DIR/$GO_BINARY"
chmod +x "$RESOURCES_DIR/$GO_BINARY"

# 3. Compile Swift menu bar app
echo "[3/4] Compiling Swift menu bar app..."
SWIFT_SRC="$PROJECT_ROOT/apps/macos-menu/main.swift"
if [ -f "$SWIFT_SRC" ]; then
  swiftc -o "$MACOS_DIR/LarkRadarMenu" \
    "$SWIFT_SRC" \
    -framework Cocoa \
    -framework Foundation

  if [ ! -f "$MACOS_DIR/LarkRadarMenu" ]; then
    echo "ERROR: Swift menu bar app compilation failed"
    exit 1
  fi
  echo "    Menu bar app compiled successfully"
else
  echo "ERROR: Swift menu source not found at apps/macos-menu/main.swift"
  exit 1
fi

# 4. Create Info.plist
echo "[4/4] Creating Info.plist..."

cat > "$CONTENTS_DIR/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>LarkRadarMenu</string>
  <key>CFBundleIdentifier</key>
  <string>$BUNDLE_ID</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>$APP_NAME</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>$VERSION</string>
  <key>CFBundleVersion</key>
  <string>1</string>
  <key>LSMinimumSystemVersion</key>
  <string>12.0</string>
  <key>LSUIElement</key>
  <true/>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

# Create empty icon file (user can replace with real icon)
touch "$RESOURCES_DIR/icon.icns"

echo ""
echo "=== Build complete ==="
echo "Output: $APP_DIR"
echo "Size: $(du -sh "$APP_DIR" | cut -f1)"
echo ""
echo "App bundle structure:"
find "$APP_DIR" -maxdepth 3 -print | sed "s|$DIST_DIR/||"
echo ""
echo "To test: open \"$APP_DIR\""
echo "To distribute: ditto -c -k --keepParent \"$APP_DIR\" \"$DIST_DIR/Lark-Radar-macOS-$ARCH.zip\""
