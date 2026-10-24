# CLAUDE.md — Endpaper (Kindle → Hardcover sync)

## Project goal
A small background daemon on a **jailbroken Kindle** that reads the user's reading
state from the **stock Amazon Kindle reader** and sends it to **Hardcover.app**.

- The user reads normally in Amazon's default reader.
- The daemon observes: current book, position (location), percent, page (if available).
- The daemon updates the user's Hardcover "currently reading" progress.

## Hard constraints (do not break)
- **No KOReader.** Do not replace or modify the Amazon reader. We only observe it.
- **Lowest common denominator.** Must work on the oldest common jailbroken Kindle,
  not only on the developer's one device. Assume fair tolerances.
- **No per-model config** unless there is no other way.
- **Simple install:** jailbreak → install package → sign in with a code → done.
  No SSH after setup.

## Design decisions (already made)
1. **Companion daemon, not a reader.** Event-driven (LIPC + inotify, see
   `docs/roadmap.md` §1), slow poll only as a safety net. Send only on change.
   Near-real-time is not required; a sync after the user closes the book is fine.
2. **Data source priority** (see `docs/kindle-data-sources.md`):
   1. Kindle local reading DB (e.g. `cc.db`) — most likely stable across firmware.
   2. Filesystem / process checks — which book is open.
   3. LIPC (`lipc-get-prop`) — extra data where available. Never required.
3. **Position = percent, from the first usable source** (user decision
   2026-09-28, `internal/syncer/position.go`):
   1. `cc.db` `p_percentFinished` (confirmed: written on go-Home and sleep);
   2. `cc.db` `p_lastAccessedPosition` ÷ text length (empty on FW 5.17.1);
   3. sidecar `.azw3f` `lpr` ÷ text length — **fallback only**, always logged
      as a cross-check. Also used (user decision 2026-10-01) at a sleep /
      go-Home when cc.db has no new value and the sidecar is newer and
      0.1–15 points ahead; never finishes a book.
   Hardcover page = percent × edition pages. **APNX page maps are dropped**
   (Calibre APNX on the device is synthetic: fixed 2300 chars per page).
4. **Capability model.** Backends are optional (`database`, `sidecar`, `lipc`,
   `inotify`); a missing one must never stop the daemon.
5. **Auth = Hardcover OAuth Device Authorization Grant.** Kindle shows a code/QR,
   user approves on phone at hardcover.app/link. No token typing on the Kindle.
6. **Runtime: one static ARM binary in Go** (`CGO_ENABLED=0`), with its own TLS
   and bundled CA certs. Do not rely on system curl/Python.
   Risk: new Go needs a newer Linux kernel than old Kindles have. Pin the Go
   version after a test binary runs on a real device (see `docs/open-questions.md`).

## Proposed layout
```
cmd/daemon/          entry point, poll loop, change detection
internal/readers/    cc.db reader
internal/sidecar/    KRDS sidecar reader (.azw3f lpr/fpr)
internal/clippings/  My Clippings.txt parser
internal/metrics/    fmcache.db reader (end-of-book star rating)
internal/syncer/     sync logic: progress, finish, clips, ratings
internal/daemon/     change detection + state
internal/events/     LIPC + inotify watchers
internal/book/       book identity (ASIN, title, author, ISBN)
internal/hardcover/  OAuth device flow, token refresh, GraphQL client
internal/config/     config + token storage
internal/store/      our state DB (SQLite)
internal/appreg/     register our page in the Kindle's appreg.db
cmd/endpaper/        the program: app (library tap), daemon, local page API,
                     ui/ (Mesquite page, embedded), dev commands
internal/certs/      bundled CA roots
packaging/kindle/    documents/Endpaper.sh (the library item)
scripts/             build scripts
docs/                context docs (read these)
```

## Where we left off
Name (user decision 2026-10-24): **Endpaper** — "a native Kindle →
hardcover.app sync". Binary, folders and library item are `endpaper`
(`/mnt/us/endpaper`, `/var/local/endpaper`, `documents/Endpaper.sh`, app id
`io.github.ipkstef.endpaper`). The first run moves an old `hcbridge`
install (`migrateLegacy`). The GitHub repo is still
`kindle-hardcover-bridge` (rename is the user's choice; module path and
URLs follow it).

Core sync is done and **works on device** (e118f27): progress (cc.db,
sleep sidecar fallback), auto-add, finish, re-read, ratings + result box,
highlights/notes, offline retry, SQLite state, self-test. History of the
research and device results: `docs/findings-bellatrix-5.17.1.md`,
`docs/roadmap.md`.

Distribution (user decisions 2026-10-02 … 10-24): **no KUAL, no KPM.** A
GitHub release zip: `documents/Endpaper.sh` (scriptlet = library item) +
`endpaper/` folder, copied by USB. The tap runs `endpaper app`: starts the
daemon (also when signed out) and opens our **Mesquite page** (registered
in `/var/local/appreg.db`; minor appreg writes are OK, user 2026-10-24).
The page talks to the daemon over 127.0.0.1 (random port + key in
`config.js`): sign in with QR, status, sync now, import old highlights,
check setup, save log, sign out, stop, uninstall. Fallback without
Mesquite: eips text screens. No autostart at boot (one tap after reboot).
Uninstall = page button, or delete the folder: the daemon sees it gone
(only while /mnt/us is mounted, never in USB mode) and removes the rest.
This build is **not yet on device**; checklist in `docs/testing.md`.

License MIT. Release: push tag `vX.Y.Z` → `.github/workflows/release.yml`
(needs repo secret `HC_CLIENT_ID`). The page shows an update banner when
GitHub has a newer release (checked at most daily, only while the page is
open; release builds only). Open: public repo + registered public client
ID, first tagged release, old-kernel device check.
Go version still open (needs an old-kernel device); builds use Go 1.23.

## Docs
- `docs/architecture.md` — data flow and sync loop
- `docs/kindle-data-sources.md` — cc.db, LIPC services, APNX, research plan
- `docs/hardcover-api.md` — GraphQL + OAuth device flow details
- `docs/open-questions.md` — unknowns and risks
- `docs/references.md` — links and prior art
- `docs/findings-bellatrix-5.17.1.md` — results from the user's Kindle
- `docs/roadmap.md` — user goals and plan to the end product
- `docs/hardcover-features.md` — journals, ratings, finish, re-read, scopes, limits
- `docs/decisions/embedded-db.md` — state DB choice (modernc sqlite, later)
- `docs/kindlemodding-notes.md` — KindleModding wiki: scriptlets, KPM, boot, appreg
- `docs/testing.md` — how we test: fixtures, self-test, one checklist per build
- `docs/decisions/telemetry.md` — opt-in telemetry to S3 (draft, user decisions open)

## Working rules for Claude
- Mark anything not verified on a real device or in real docs as **UNVERIFIED**.
- Prefer small, testable steps. The user has one Kindle for testing.
- Keep answers short. Use simple technical English.
