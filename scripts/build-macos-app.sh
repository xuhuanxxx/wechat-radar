#!/bin/bash
set -e

# Build Lark Radar as a macOS .app bundle with menu bar
# Usage: ./scripts/build-macos-app.sh
# Output: dist/Lark Radar.app

APP_NAME="Lark Radar"
BUNDLE_ID="com.lark-radar.app"
VERSION="0.1.0"

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
  NODE_ARCH="darwin-arm64"
else
  NODE_ARCH="darwin-x64"
fi

# Paths
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST_DIR="$PROJECT_ROOT/dist"
APP_DIR="$DIST_DIR/$APP_NAME.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"
NODE_DIR="$RESOURCES_DIR/node"
APP_RES_DIR="$RESOURCES_DIR/app"

NODE_VERSION="v20.18.0"
NODE_TARBALL="node-$NODE_VERSION-$NODE_ARCH.tar.gz"
NODE_URL="https://nodejs.org/dist/$NODE_VERSION/$NODE_TARBALL"

echo "=== Building $APP_NAME for $ARCH ==="

# Clean and create directories
rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR" "$APP_RES_DIR"

# 1. Build Next.js app with standalone output
echo "[1/9] Building Next.js app..."
cd "$PROJECT_ROOT"
pnpm build

# 2. Download and extract Node.js binary
echo "[2/9] Downloading Node.js $NODE_ARCH..."
mkdir -p "$DIST_DIR"
if [ ! -f "$DIST_DIR/$NODE_TARBALL" ]; then
  echo "    Downloading from $NODE_URL..."
  curl -L -o "$DIST_DIR/$NODE_TARBALL" "$NODE_URL"
fi

echo "    Extracting Node.js..."
tar -xzf "$DIST_DIR/$NODE_TARBALL" -C "$RESOURCES_DIR"
mv "$RESOURCES_DIR/node-$NODE_VERSION-$NODE_ARCH" "$NODE_DIR"

if [ ! -f "$NODE_DIR/bin/node" ]; then
  echo "ERROR: Node.js binary not found after extraction"
  exit 1
fi

# Create renamed node binary for better process naming in Activity Monitor
echo "    Creating renamed node binary..."
cp "$NODE_DIR/bin/node" "$NODE_DIR/bin/LarkRadarServer"

# 3. Copy standalone output
echo "[3/9] Copying standalone output..."
cd "$PROJECT_ROOT/.next/standalone"
tar -chf - . 2>/dev/null | tar -xf - -C "$APP_RES_DIR/"
cd "$PROJECT_ROOT"

if [ -d "$PROJECT_ROOT/.next/static" ]; then
  mkdir -p "$APP_RES_DIR/.next/static"
  cp -r "$PROJECT_ROOT/.next/static/"* "$APP_RES_DIR/.next/static/" 2>/dev/null || true
fi

# 4. Install production dependencies with embedded Node
echo "[4/9] Installing production dependencies..."
export PATH="$NODE_DIR/bin:$PATH"
cd "$APP_RES_DIR"
cp "$PROJECT_ROOT/package.json" .
rm -rf node_modules
"$NODE_DIR/bin/npm" install --omit=dev --no-audit --no-fund 2>&1 | tail -5

# 5. Rebuild better-sqlite3 for the embedded Node version
echo "[5/9] Rebuilding better-sqlite3 for Node $NODE_VERSION..."
rm -rf "$APP_RES_DIR/node_modules/better-sqlite3"
cp -r "$PROJECT_ROOT/node_modules/better-sqlite3" "$APP_RES_DIR/node_modules/"
cd "$APP_RES_DIR/node_modules/better-sqlite3"
"$NODE_DIR/bin/npm" install --production --no-audit --no-fund 2>&1 | tail -3
"$NODE_DIR/bin/npm" rebuild 2>&1 | tail -5
cd "$APP_RES_DIR"

# Copy the rebuilt binary to the standalone-traced location
STANDALONE_BS=$(find "$APP_RES_DIR/.next/node_modules" -name "better_sqlite3.node" -type f 2>/dev/null | head -1)
if [ -n "$STANDALONE_BS" ]; then
  cp "$APP_RES_DIR/node_modules/better-sqlite3/build/Release/better_sqlite3.node" "$STANDALONE_BS"
fi

# Verify the binary was built
if [ ! -f "$APP_RES_DIR/node_modules/better-sqlite3/build/Release/better_sqlite3.node" ]; then
  echo "ERROR: better-sqlite3 binary not found after rebuild"
  exit 1
fi

echo "    better-sqlite3 rebuilt successfully"

# 6. Optimize bundle size
echo "[6/9] Optimizing bundle size..."
cd "$APP_RES_DIR"

# Strip source maps
find node_modules -name "*.map" -type f -delete 2>/dev/null || true

# Remove TypeScript declaration files
find node_modules -name "*.d.ts" -type f -delete 2>/dev/null || true
find node_modules -name "*.d.mts" -type f -delete 2>/dev/null || true

# Remove documentation files from top-level packages
find node_modules -maxdepth 2 -name "README*" -type f -delete 2>/dev/null || true
find node_modules -maxdepth 2 -name "LICENSE*" -type f -delete 2>/dev/null || true
find node_modules -maxdepth 2 -name "CHANGELOG*" -type f -delete 2>/dev/null || true
find node_modules -maxdepth 2 -name "*.md" -type f -delete 2>/dev/null || true

cd "$NODE_DIR"
# Remove Node.js development files (only bin/node is needed at runtime)
rm -rf include lib share CHANGELOG.md README.md LICENSE 2>/dev/null || true

echo "    Optimization complete"

# 7. Compile Swift menu bar app
echo "[7/9] Compiling menu bar app..."
if [ -f "$PROJECT_ROOT/scripts/macos-menu/main.swift" ]; then
  swiftc -o "$MACOS_DIR/LarkRadarMenu" "$PROJECT_ROOT/scripts/macos-menu/main.swift" -framework Cocoa 2>&1
  if [ ! -f "$MACOS_DIR/LarkRadarMenu" ]; then
    echo "WARNING: Menu bar app compilation failed, falling back to launch script"
    cp "$PROJECT_ROOT/scripts/macos-menu/main.swift" "$MACOS_DIR/"
  else
    echo "    Menu bar app compiled successfully"
  fi
else
  echo "WARNING: Swift menu source not found, using fallback launch script"
fi

# 8. Create fallback launch script (for manual/server mode)
echo "[8/9] Creating fallback launch script..."
cat > "$MACOS_DIR/launch" << 'SCRIPT'
#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
RESOURCES_DIR="$(dirname "$SCRIPT_DIR")/Resources"
NODE_BIN="$RESOURCES_DIR/node/bin/node"
APP_DIR="$RESOURCES_DIR/app"
CONFIG_DIR="$HOME/.lark-radar"
CONFIG_FILE="$CONFIG_DIR/config.json"

mkdir -p "$CONFIG_DIR"

# Read port from config, default to 3456
PORT=3456
if [ -f "$CONFIG_FILE" ]; then
  PORT=$("$NODE_BIN" -e "
    const fs = require('fs');
    try {
      const cfg = JSON.parse(fs.readFileSync('$CONFIG_FILE', 'utf-8'));
      const p = parseInt(cfg.port, 10);
      console.log(p >= 1024 && p <= 65535 ? p : 3456);
    } catch { console.log(3456); }
  " 2>/dev/null || echo 3456)
fi

# Find available port
while lsof -i :"$PORT" -t >/dev/null 2>&1; do
  PORT=$((PORT + 1))
  if [ "$PORT" -gt 65535 ]; then
    PORT=3456
  fi
done

echo "Starting Lark Radar on port $PORT..."

export PORT="$PORT"
export LARK_RADAR_DATA_DIR="$CONFIG_DIR"

cd "$APP_DIR"
"$NODE_BIN" "$APP_DIR/server.js" &
SERVER_PID=$!

# Wait for server
for i in {1..30}; do
  if curl -s "http://localhost:$PORT/api/setup" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

open "http://localhost:$PORT"

wait $SERVER_PID
SCRIPT

chmod +x "$MACOS_DIR/launch"

# 9. Create Info.plist
echo "[9/9] Creating Info.plist..."

# Use menu bar app as main executable if available, otherwise fallback to launch script
if [ -f "$MACOS_DIR/LarkRadarMenu" ]; then
  EXECUTABLE="LarkRadarMenu"
  LSUI_ELEMENT="<true/>"
else
  EXECUTABLE="launch"
  LSUI_ELEMENT="<false/>"
fi

cat > "$CONTENTS_DIR/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleExecutable</key>
  <string>$EXECUTABLE</string>
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
  $LSUI_ELEMENT
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST

touch "$RESOURCES_DIR/icon.icns"

echo ""
echo "=== Build complete ==="
echo "Output: $APP_DIR"
echo "Size: $(du -sh "$APP_DIR" | cut -f1)"
echo "Executable: $EXECUTABLE"
echo ""
echo "To test: open \"$APP_DIR\""
echo "To distribute: ditto -c -k --keepParent \"$APP_DIR\" \"$DIST_DIR/Lark-Radar-macOS-$ARCH.zip\""
