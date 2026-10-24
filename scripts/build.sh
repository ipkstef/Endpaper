#!/bin/sh
# Build the release zip: dist/hcbridge-<version>.zip
# Unzip it onto the Kindle's USB drive: it holds documents/Hardcover.sh (the
# library item) and the hcbridge folder.
# Built with Go 1.23 (lowest kernel floor, 2.6.32). See docs/open-questions.md.
# HC_CLIENT_ID=... bakes the public OAuth client ID into the binary; without
# it, a client_id.txt in the hcbridge folder is needed.
set -eu
cd "$(dirname "$0")/.."
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
OUT=dist/pkg
rm -rf "$OUT" && mkdir -p "$OUT/hcbridge/bin" "$OUT/documents"
GOTOOLCHAIN=go1.23.12 CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 \
	go build -trimpath -ldflags "-s -w -X main.clientID=${HC_CLIENT_ID:-} -X main.version=$VERSION" \
	-o "$OUT/hcbridge/bin/hcbridge" ./cmd/hcbridge
cp packaging/kindle/documents/Hardcover.sh "$OUT/documents/"
cp README.md "$OUT/hcbridge/README.md"
[ -f LICENSE ] && cp LICENSE "$OUT/hcbridge/LICENSE"
# Local builds: a client_id.txt next to the package is copied in.
[ -f packaging/kindle/hcbridge/client_id.txt ] && cp packaging/kindle/hcbridge/client_id.txt "$OUT/hcbridge/"
ZIP="dist/hcbridge-$VERSION.zip"
rm -f "$ZIP"
(cd "$OUT" && zip -qr "../$(basename "$ZIP")" documents hcbridge)
rm -rf "$OUT"
echo "built $ZIP"
