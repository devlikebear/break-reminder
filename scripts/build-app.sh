#!/usr/bin/env bash
# Builds and signs the menu bar .app with the Go engine sidecar.
#
#   scripts/build-app.sh             Developer ID build (hardened runtime)
#   scripts/build-app.sh --sandbox   same identity + App Sandbox entitlements,
#                                    for local sandbox verification (spike)
#
# SIGN_IDENTITY overrides the identity ("-" signs ad hoc). By default the first
# "Developer ID Application" identity in the keychain is used, falling back to
# ad hoc when there is none. TIMESTAMP=0 skips the secure timestamp (offline).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

SANDBOX=0
[[ "${1:-}" == "--sandbox" ]] && SANDBOX=1

VERSION="$(cat VERSION 2>/dev/null || echo dev)"
BUILD="$(git rev-list --count HEAD 2>/dev/null || echo 1)"
OUT_DIR="$ROOT/bin"
APP_NAME="Break Reminder"
[[ $SANDBOX == 1 ]] && APP_NAME="Break Reminder (Sandbox)"
APP="$OUT_DIR/$APP_NAME.app"
RES="$ROOT/helpers/Resources"
SWIFT_OUT="$ROOT/helpers/.build/release"

if [[ -z "${SIGN_IDENTITY:-}" ]]; then
  SIGN_IDENTITY="$(security find-identity -v -p codesigning 2>/dev/null \
    | sed -n 's/.*"\(Developer ID Application: [^"]*\)".*/\1/p' | head -1)"
  SIGN_IDENTITY="${SIGN_IDENTITY:--}"
fi

echo "==> Building engine ($VERSION)"
mkdir -p "$OUT_DIR"
go build -ldflags "-s -w -X main.version=$VERSION" -o "$OUT_DIR/engine" ./cmd/break-reminder

echo "==> Building Swift helpers"
(cd helpers && swift build -c release --product BreakReminderApp && swift build -c release --product BreakScreenApp)

echo "==> Assembling $APP"
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
sed -e "s/__VERSION__/$VERSION/" -e "s/__BUILD__/$BUILD/" "$RES/App-Info.plist" > "$APP/Contents/Info.plist"
cp "$SWIFT_OUT/BreakReminderApp" "$APP/Contents/MacOS/BreakReminderApp"
cp "$OUT_DIR/engine" "$APP/Contents/MacOS/engine"
cp "$SWIFT_OUT/BreakScreenApp" "$APP/Contents/MacOS/break-screen"
cp -R "$SWIFT_OUT/BreakReminderHelpers_BreakScreenApp.bundle" "$APP/Contents/Resources/"

if [[ $SANDBOX == 1 ]]; then
  APP_ENT="$RES/app-sandbox.entitlements"
  CHILD_ENT="$RES/engine-sandbox.entitlements"
else
  APP_ENT="$RES/app.entitlements"
  CHILD_ENT="$RES/engine.entitlements"
fi

SIGN_ARGS=(--force --options runtime)
if [[ "$SIGN_IDENTITY" != "-" && "${TIMESTAMP:-1}" != "0" ]]; then
  SIGN_ARGS+=(--timestamp)
fi

echo "==> Signing with: $SIGN_IDENTITY"
# Inside-out: nested code first, the app bundle last.
codesign "${SIGN_ARGS[@]}" --sign "$SIGN_IDENTITY" --entitlements "$CHILD_ENT" "$APP/Contents/MacOS/engine"
codesign "${SIGN_ARGS[@]}" --sign "$SIGN_IDENTITY" --entitlements "$CHILD_ENT" "$APP/Contents/MacOS/break-screen"
codesign "${SIGN_ARGS[@]}" --sign "$SIGN_IDENTITY" --entitlements "$APP_ENT" "$APP"

codesign --verify --deep --strict "$APP"
echo "==> Built $APP"
