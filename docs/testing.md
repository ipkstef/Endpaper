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
- **Check setup** (Endpaper page): checks the page, Mesquite, tools,
  inotify, free space, state DB, cc.db, latest book + sidecar, ratings
  file, clippings, Hardcover sign-in and the latest book's match
  (read-only). Result on the page and in `endpaper/endpaper-selftest.txt`.
- **Save log for support** (Endpaper page): header (version, daemon pid,
  DB counts, last check) + log, in `endpaper/endpaper-log.txt`. Each sleep /
  go-Home logs one line (update, or "no new position in cc.db … sidecar
  …"); other events are summed up hourly.
- **One checklist per build**: every build comes with one complete list of
  steps and the log lines that mean pass.

## Checklist: Endpaper scriptlet + Mesquite page build (2026-10-24)
Result v0.1.0 (device log 2026-10-24 07:36): steps 0, 1, 2, 3, 5, 6, 8
**pass** (migration moved 9 files and kept the state; page opened as
window `L:A_N:application_PC:TS_ID:io.github.ipkstef.endpaper`; page
reached the API; sign-out, then sign-in with code in 31 s; check setup all
OK). Second log (07:45): 7 **pass** (page reopened twice, no daemon restart);
highlights/notes sent. Bug: after "Sync now", the message stayed (an
unchanged scan did not save "Last check", and the page never cleared the
message) → fixed for v0.1.1. Open: 4 (retest on v0.1.1), 9–12.

Setup: unzip `endpaper-<version>.zip` onto the drive (do not delete the
old KUAL `extensions/hcbridge` yourself). Copy your `client_id.txt` into
the new `endpaper` folder. Eject.

0. **Migration:** the first tap moves the old install. Log:
   `migrate from hcbridge: stopped old daemon …`, `moved N files …`,
   `removed /mnt/us/extensions/hcbridge`. Pass: still signed in (step 3),
   and the old KUAL "Hardcover" menu is gone.
1. **Library item:** "Endpaper" shows in the library. Tap it.
   Pass: the Endpaper page opens within ~10 s (black/white, buttons).
   Log: `launch: registered io.github.ipkstef.endpaper in appreg.db`,
   `launch: page started, active window "…"`,
   `page api: first request from the page`.
   If no page opens: the text screen (status or sign-in code) must show
   instead; send the log.
2. **Daemon:** log `daemon: started, version …, page api: 127.0.0.1:…`
   (only one daemon: the old one was stopped in step 0).
3. **Status:** page shows "Signed in as @you", last check, last result
   (you are already signed in).
4. **Sync now:** tap it. Pass: "Checking your books now.", then within
   ~10 s "Done. …" and "Last check" changes to now; the message clears
   after ~20 s, and is gone when you leave and come back. Log (v0.1.1+):
   `daemon: scan (sync now): …`.
4b. **Cover (v0.1.1+):** the library item shows the Endpaper logo (may
   need a library refresh). UNVERIFIED: the icon header on this firmware.
5. **Check setup:** list of OK/FAIL lines on the page. Send a photo if any
   FAIL.
6. **Sign out → Sign in:** Sign out (confirm). Page shows "Sign in". Tap
   Sign in: QR code + code appear. Scan the QR with your phone, approve.
   Pass: within ~10 s the page shows "Signed in as @you". Log `login: OK`.
7. **Leave and return:** Home button, read a page, go Home, tap Endpaper.
   Pass: page opens fast (no restart: no new `daemon: started` line);
   last result shows the book.
8. **Save log for support:** connect USB; `endpaper/endpaper-log.txt`
   exists.
9. **USB mode is safe:** while USB is connected, the folder looks gone.
   After eject, tap Endpaper: still signed in (no `uninstall:` line).
10. **Stop sync:** tap Stop (confirm). Home, tap Endpaper again: sync
    starts again (`daemon: started`).
11. **Uninstall by folder delete:** with USB, delete `documents/Endpaper.sh`
    and the `endpaper` folder. Eject. Open and close a book (wakes the
    daemon). Pass: log gone; with SSH/next install: `/var/local/endpaper`
    and `/var/local/mesquite/endpaper` gone. Easy check: reinstall the zip,
    tap Endpaper → page asks you to **sign in** again (state was removed).
12. **Uninstall button:** tap Uninstall (confirm). Pass: page says removed;
    after Home, the "Endpaper" item is gone from the library (may need a
    library refresh); USB shows no `endpaper` folder.
13. Reinstall for normal use.
14. **Update banner** (only after the first tagged release, with a release
    build older than it): the page shows "Update available: vX.Y.Z".
