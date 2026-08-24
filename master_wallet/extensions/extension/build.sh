#!/usr/bin/env bash
# Build a per-browser extension package from the single canonical source.
# Usage: ./build.sh <chrome|brave|edge|firefox|safari> [output-dir]
set -u

BROWSER="${1:-chrome}"
OUT="${2:-dist/$BROWSER}"
SRC="$(cd "$(dirname "$0")" && pwd)"

case "$BROWSER" in
  chrome|brave|edge|firefox|safari) ;;
  *) echo "unknown browser: $BROWSER (expected chrome|brave|edge|firefox|safari)" >&2; exit 1 ;;
esac

rm -rf "$OUT"
mkdir -p "$OUT"

# Shared sources are byte-identical across browsers — copy them once.
cp -r "$SRC/background.js" "$SRC/injected.js" "$SRC/popup.html" "$SRC/popup.js" "$SRC/services" "$OUT/"
cp "$SRC"/icon*.png "$OUT/" 2>/dev/null || true

# Chrome uses the canonical MV3 manifest; other browsers use their variant.
if [ "$BROWSER" = "chrome" ]; then
  cp "$SRC/manifest.json" "$OUT/manifest.json"
else
  cp "$SRC/manifests/manifest.$BROWSER.json" "$OUT/manifest.json"
fi

# Safari additionally requires Xcode conversion to an app extension:
#   xcrun safari-web-extension-converter "$OUT" --app-name "TigerMasterWallet"
echo "built $BROWSER extension -> $OUT"
