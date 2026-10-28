#!/bin/sh
# Name: Endpaper Probe
# Author: Endpaper (research tool)
# DontUseFBInk
#
# Read-only research for a Hardcover home screen: collects how the home
# screen is built (about 1 min), then watches for 3 minutes while you use
# the home screen. Writes only to /mnt/us/endpaper-probe. Delete this file
# and the endpaper-probe-bin folder when done.
DIR=/mnt/us/endpaper-probe-bin
BIN=/tmp/endpaper-homeprobe
if [ ! -f "$DIR/homeprobe" ]; then
	eips 1 3 "Endpaper probe: folder endpaper-probe-bin missing."
	exit 1
fi
cp "$DIR/homeprobe" "$BIN.new" && chmod 755 "$BIN.new" && mv -f "$BIN.new" "$BIN" || exit 1
S=""
command -v setsid >/dev/null 2>&1 && S=setsid
$S nohup "$BIN" >/mnt/us/endpaper-probe-run.txt 2>&1 &
exit 0
