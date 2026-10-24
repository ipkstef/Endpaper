# Architecture

Endpaper is one static ARM binary (`cmd/endpaper`, Go, `CGO_ENABLED=0`) with
a small web page for the Kindle. It only reads the Kindle's files and talks
to Hardcover's GraphQL API.

## Parts on the Kindle

```
 Library item  documents/Endpaper.sh ── tap ──▶ endpaper app
                                                  │ starts (if needed)
                                                  ▼
 ┌──────────────────────── endpaper daemon ────────────────────────┐
 │ events ─▶ debounce ─▶ scan ─▶ syncer ─▶ Hardcover GraphQL        │
 │                         │                                         │
 │                    state DB (SQLite)        local page API        │
 │                                             127.0.0.1:<random>    │
 └───────────────────────────────────────────────────▲──────────────┘
                                                     │ XHR + key
                         Mesquite page  /var/local/mesquite/endpaper
                         (registered in /var/local/appreg.db)
```

| Path | What | Why there |
|---|---|---|
| `/mnt/us/documents/Endpaper.sh` | library item (scriptlet) | the user taps it |
| `/mnt/us/endpaper/` | binary, README, LICENSE | USB-visible; deleting it uninstalls |
| `/tmp/endpaper` | running copy of the binary | `/mnt/us` is FAT (no exec), RAM |
| `/var/local/endpaper/` | token, `endpaper.db`, log | ext3, kept across reboots and USB mode |
| `/var/local/mesquite/endpaper/` | the page + `config.js` (port, key) | readable by Mesquite |

## Triggers (event driven)

| Event | Source | Meaning |
|---|---|---|
| `appPaused` reader | LIPC `com.lab126.appmgrd` | user left the book |
| `goingToScreenSaver` | LIPC `com.lab126.powerd` | sleep (also writes the position) |
| `connectionAvailable` | LIPC `com.lab126.cmd` | Wi-Fi back: send what waits |
| `cc.db` journal deleted / WAL written | inotify | the Kindle committed reading data |
| `fmcache.db` commit | inotify | a star rating was tapped (sent at once) |
| Sync now / sign-in | page API | user action |
| safety poll, 15 min | timer | skipped while the reader is open |

Triggers within 5 s join one scan. Each source is optional: a missing one
never stops the daemon (capability model).

## One scan

1. Read all books from `cc.db` (read-only) and compare with the snapshot.
2. For each changed book, oldest first: **identify** it on Hardcover
   (waterfall: ASIN → ISBN → own library → search; never guess), then
   **sync**: progress (forward only), auto-add, finish (> 99 % or read
   state 2), re-read.
3. Then ratings (`fmcache.db`) and highlights/notes (`My Clippings.txt`).
4. Save the state by diff (only changed rows; "Last check" at most hourly
   unless the user asked).

Position source order: `cc.db` percent → `cc.db` position ÷ length →
sidecar `.azw3f` (fallback, never finishes a book). Details:
`docs/research/findings-bellatrix-5.17.1.md`.

All Hardcover writes run in one goroutine. Offline (DNS or dial errors) the
daemon stops trying until Wi-Fi is back; changes wait in the state DB.

## Packages

| Package | Job |
|---|---|
| `cmd/endpaper` | commands, daemon loop, page API, launcher, uninstall |
| `cmd/endpaper/ui` | the Mesquite page (ES5 HTML/JS, embedded in the binary) |
| `internal/readers` | `cc.db` reader |
| `internal/sidecar` | KRDS sidecar reader (`.azw3f` `lpr`) |
| `internal/mobi` | EXTH metadata (ISBN, ASIN) from book files |
| `internal/book` | book identity, title rules |
| `internal/match` | the identify waterfall |
| `internal/syncer` | progress, finish, re-read, ratings, clips |
| `internal/daemon` | change detection, retries, state |
| `internal/store` | our SQLite state DB (diff writes) |
| `internal/hardcover` | OAuth device flow, token refresh, GraphQL client |
| `internal/metrics` | `fmcache.db` (end-of-book ratings) |
| `internal/kindledb` | retry for reads of the Kindle's DBs during its commits |
| `internal/clippings` | `My Clippings.txt` parser |
| `internal/events` | LIPC and inotify watchers |
| `internal/appreg` | our rows in the Kindle's `appreg.db` |
| `internal/screen` | eips text, pillow alert box |
| `internal/atomicfile`, `internal/config`, `internal/certs` | safe writes, paths + token, bundled CA roots |

## Install, update, uninstall

- **Install:** unzip the release onto the drive, tap Endpaper, sign in on
  the page (device code + QR).
- **Update:** copy the new files (over the old ones or after deleting
  them), tap Endpaper: a daemon of another version is replaced, and an
  open page is closed so a fresh one loads. The page shows a banner when
  GitHub has a newer release (checked at most daily, only while the page
  is open).
- **Uninstall:** only the page button. Endpaper never deletes anything on
  its own: on MTP Kindles `/mnt/us` stays mounted during USB, so a
  "folder deleted" rule uninstalled during a normal update (v0.1.0/v0.1.1,
  device 2026-10-24).
- No autostart at boot (needs a rootfs write): one tap after a reboot.
