#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

DRY_RUN=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    *) echo "unknown flag: $arg" >&2; exit 1 ;;
  esac
done

VERSION=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' apps/desktop/build/Info.plist)
APP="dist/Tiles Spliter.app"

mkdir -p dist

# 1. Frontend
(cd apps/desktop/frontend && bun install && bun run build)

# 2. Universal binary
if [ "$DRY_RUN" = "1" ]; then
  # Dry run: build arm64 only (fast local verification, no cross-toolchain
  # requirement). The real release build still produces a universal binary.
  (cd apps/desktop && CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o ../../dist/tilespliter-arm64 .)
  cp dist/tilespliter-arm64 dist/tilespliter
else
  # CGO_ENABLED=1 is required explicitly: cgo defaults OFF when GOARCH differs
  # from the host, which would silently drop the AX/Carbon platform layer.
  (cd apps/desktop && CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o ../../dist/tilespliter-arm64 . \
    && CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -o ../../dist/tilespliter-amd64 .)
  lipo -create -output dist/tilespliter dist/tilespliter-arm64 dist/tilespliter-amd64
fi

# 3. Bundle
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp apps/desktop/build/Info.plist "$APP/Contents/"
cp dist/tilespliter "$APP/Contents/MacOS/Tiles Spliter"
cp apps/desktop/build/icon.icns "$APP/Contents/Resources/"

if [ "$DRY_RUN" = "1" ]; then
  echo "dry-run: skipping codesign, notarization, staple, and DMG creation"
  echo "dry-run: bundle assembled at $APP"
  exit 0
fi

# 4. Sign (hardened runtime)
codesign --force --options runtime --timestamp \
  --entitlements apps/desktop/build/entitlements.plist \
  --sign "$CODESIGN_IDENTITY" "$APP"

# 5. Notarize + staple
ditto -c -k --keepParent "$APP" dist/notarize.zip
if [ -n "${NOTARY_KEYCHAIN_PROFILE:-}" ]; then
  # Preferred: credentials stored once via `xcrun notarytool store-credentials`
  # keep the app-specific password out of argv/env.
  xcrun notarytool submit dist/notarize.zip --keychain-profile "$NOTARY_KEYCHAIN_PROFILE" --wait
else
  # Fallback: explicit Apple ID credentials. Consider switching to a keychain
  # profile (set NOTARY_KEYCHAIN_PROFILE) so the password never hits argv.
  xcrun notarytool submit dist/notarize.zip --apple-id "$APPLE_ID" --team-id "$APPLE_TEAM_ID" --password "$APPLE_APP_PASSWORD" --wait
fi
xcrun stapler staple "$APP"

# 6. DMG
hdiutil create -volname "Tiles Spliter" -srcfolder "$APP" -ov -format UDZO "dist/Tiles Spliter-$VERSION.dmg"
echo "done: dist/Tiles Spliter-$VERSION.dmg"
