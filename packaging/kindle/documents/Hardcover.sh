#!/bin/sh
# Name: Hardcover
# Author: kindle-hardcover-bridge
# DontUseFBInk
#
# Tap "Hardcover" in the library: starts background sync and opens the
# Hardcover page (sign in, status, settings). Uninstall: use the page's
# Uninstall button, or delete this file and the "hcbridge" folder.
HOME_DIR=/mnt/us/hcbridge
BIN=/tmp/hcbridge

if [ ! -f "$HOME_DIR/bin/hcbridge" ]; then
	eips 1 3 "Hardcover: the hcbridge folder is missing."
	eips 1 4 "Copy it to the Kindle again (see README)."
	exit 1
fi
# /mnt/us is FAT (no exec bit): run a copy from /tmp. Copy, then rename:
# the daemon may be running from $BIN ("text file busy").
cp "$HOME_DIR/bin/hcbridge" "$BIN.new" && chmod 755 "$BIN.new" && mv -f "$BIN.new" "$BIN" || exit 1
CID=""
[ -f "$HOME_DIR/client_id.txt" ] && CID="$(tr -d ' \r\n' < "$HOME_DIR/client_id.txt")"
# Detach: the launcher may stop the script's process group when it ends.
S=""
command -v setsid >/dev/null 2>&1 && S=setsid
$S nohup "$BIN" app -home "$HOME_DIR" ${CID:+-client-id "$CID"} >/dev/null 2>&1 &
exit 0
