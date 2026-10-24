#!/bin/sh
# Name: Endpaper
# Author: Endpaper (github.com/ipkstef/endpaper)
# DontUseFBInk
#
# Tap "Endpaper" in the library: starts background sync and opens the
# Endpaper page (sign in, status, settings). Uninstall: use the page's
# Uninstall button, or delete this file and the "endpaper" folder.
HOME_DIR=/mnt/us/endpaper
BIN=/tmp/endpaper

if [ ! -f "$HOME_DIR/bin/endpaper" ]; then
	eips 1 3 "Endpaper: the endpaper folder is missing."
	eips 1 4 "Copy it to the Kindle again (see README)."
	exit 1
fi
# /mnt/us is FAT (no exec bit): run a copy from /tmp. Copy, then rename:
# the daemon may be running from $BIN ("text file busy").
cp "$HOME_DIR/bin/endpaper" "$BIN.new" && chmod 755 "$BIN.new" && mv -f "$BIN.new" "$BIN" || exit 1
CID=""
[ -f "$HOME_DIR/client_id.txt" ] && CID="$(tr -d ' \r\n' < "$HOME_DIR/client_id.txt")"
# Detach: the launcher may stop the script's process group when it ends.
S=""
command -v setsid >/dev/null 2>&1 && S=setsid
$S nohup "$BIN" app -home "$HOME_DIR" ${CID:+-client-id "$CID"} >/dev/null 2>&1 &
exit 0
