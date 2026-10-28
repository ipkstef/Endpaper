#!/bin/sh
# Build the home screen research probe: dist/endpaper-homeprobe.zip
# (not a release; unzip onto the Kindle drive, tap "Endpaper Probe").
set -eu
cd "$(dirname "$0")/.."
OUT=dist/probe
rm -rf "$OUT" && mkdir -p "$OUT/endpaper-probe-bin" "$OUT/documents"
GOTOOLCHAIN=go1.23.12 CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 \
	go build -trimpath -ldflags "-s -w" -o "$OUT/endpaper-probe-bin/homeprobe" ./cmd/homeprobe
cp packaging/homeprobe/documents/EndpaperProbe.sh "$OUT/documents/"
rm -f dist/endpaper-homeprobe.zip
(cd "$OUT" && zip -qr ../endpaper-homeprobe.zip documents endpaper-probe-bin)
rm -rf "$OUT"
echo "built dist/endpaper-homeprobe.zip"
