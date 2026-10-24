#!/bin/sh
# Build the release zip: dist/endpaper-<version>.zip
# Unzip it onto the Kindle's USB drive: it holds documents/Endpaper.sh (the
# library item) and the endpaper folder.
# Built with Go 1.23 (lowest kernel floor, 2.6.32). See docs/open-questions.md.
# HC_CLIENT_ID=... bakes the public OAuth client ID into the binary; without
# it, a client_id.txt in the endpaper folder is needed.
set -eu
cd "$(dirname "$0")/.."
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
OUT=dist/pkg
rm -rf "$OUT" && mkdir -p "$OUT/endpaper/bin" "$OUT/documents"
GOTOOLCHAIN=go1.23.12 CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 \
	go build -trimpath -ldflags "-s -w -X main.clientID=${HC_CLIENT_ID:-} -X main.version=$VERSION" \
	-o "$OUT/endpaper/bin/endpaper" ./cmd/endpaper
cp packaging/kindle/documents/Endpaper.sh "$OUT/documents/"
cp README.md "$OUT/endpaper/README.md"
[ -f LICENSE ] && cp LICENSE "$OUT/endpaper/LICENSE"
# Local builds: a client_id.txt next to the package is copied in.
[ -f packaging/kindle/endpaper/client_id.txt ] && cp packaging/kindle/endpaper/client_id.txt "$OUT/endpaper/"
ZIP="dist/endpaper-$VERSION.zip"
rm -f "$ZIP"
(cd "$OUT" && zip -qr "../$(basename "$ZIP")" documents endpaper)
rm -rf "$OUT"
echo "built $ZIP"
