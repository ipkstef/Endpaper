# Testing

Goal: few device rounds. The user has one Kindle.

## On the build machine
- `go test -race ./...` before every build.
- **Replay fixtures** from the user's device (FW 5.17.1):
  - `internal/readers/testdata/cc-fw5.17.1.db` — real `Entries` schema, 3
    books (collation `icu` removed).
  - `internal/sidecar/testdata/*.azw3f` — Red Rising sidecar before and
    after a sleep (sleep research 2026-09-28).
  Add new device files here when a log shows a new format.

## On the device
- **Check setup** (Hardcover page): checks the page, Mesquite, tools,
  inotify, free space, state DB, cc.db, latest book + sidecar, ratings
  file, clippings, Hardcover sign-in and the latest book's match
  (read-only). Result on the page and in `hcbridge/hcbridge-selftest.txt`.
- **Save log for support** (Hardcover page): header (version, daemon pid,
  DB counts, last check) + log, in `hcbridge/hcbridge-log.txt`. Each sleep /
  go-Home logs one line (update, or "no new position in cc.db … sidecar
  …"); other events are summed up hourly.
- **One checklist per build**: every build comes with one complete list of
  steps and the log lines that mean pass.

## Checklist: scriptlet + Mesquite page build (2026-10-24)
Setup: with USB, delete `extensions/hcbridge` (old KUAL install). Unzip
`hcbridge-<version>.zip` onto the drive. Copy your `client_id.txt` into
the new `hcbridge` folder. Eject.

1. **Library item:** "Hardcover" shows in the library. Tap it.
   Pass: the Hardcover page opens within ~10 s (black/white, buttons).
   Log: `launch: registered io.github.ipkstef.hcbridge in appreg.db`,
   `launch: page started, active window "…"`,
   `page api: first request from the page`.
   If no page opens: the text screen (status or sign-in code) must show
   instead; send the log.
2. **Old daemon replaced:** log `daemon: replacing running daemon` (if the
   KUAL one ran), then `daemon: started, version …, page api: 127.0.0.1:…`.
3. **Status:** page shows "Signed in as @you", last check, last result
   (you are already signed in).
4. **Sync now:** tap it. Pass: message "Checking your books now."; log
   `daemon: scan (sync now)`.
5. **Check setup:** list of OK/FAIL lines on the page. Send a photo if any
   FAIL.
6. **Sign out → Sign in:** Sign out (confirm). Page shows "Sign in". Tap
   Sign in: QR code + code appear. Scan the QR with your phone, approve.
   Pass: within ~10 s the page shows "Signed in as @you". Log `login: OK`.
7. **Leave and return:** Home button, read a page, go Home, tap Hardcover.
   Pass: page opens fast (no restart: no new `daemon: started` line);
   last result shows the book.
8. **Save log for support:** connect USB; `hcbridge/hcbridge-log.txt`
   exists.
9. **USB mode is safe:** while USB is connected, the folder looks gone.
   After eject, tap Hardcover: still signed in (no `uninstall:` line).
10. **Stop sync:** tap Stop (confirm). Home, tap Hardcover again: sync
    starts again (`daemon: started`).
11. **Uninstall by folder delete:** with USB, delete `documents/Hardcover.sh`
    and the `hcbridge` folder. Eject. Open and close a book (wakes the
    daemon). Pass: log gone; with SSH/next install: `/var/local/hcbridge`
    and `/var/local/mesquite/hcbridge` gone. Easy check: reinstall the zip,
    tap Hardcover → page asks you to **sign in** again (state was removed).
12. **Uninstall button:** tap Uninstall (confirm). Pass: page says removed;
    after Home, the "Hardcover" item is gone from the library (may need a
    library refresh); USB shows no `hcbridge` folder.
13. Reinstall for normal use.
