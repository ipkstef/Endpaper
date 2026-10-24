# CLAUDE.md

Guidance for AI coding assistants (and a quick orientation for human
contributors) working on **Endpaper**, a native Kindle → Hardcover.app sync.

## What Endpaper is

A small background program for **jailbroken Kindles**. It reads the reading
state that the **stock Amazon Kindle reader** already keeps on the device and
sends it to [Hardcover](https://hardcover.app): progress, new books, finished
books, re-reads, star ratings, and highlights/notes (as private journal
entries).

User-facing docs: `README.md`. How it works: `docs/architecture.md`.

## Hard constraints

Do not break these without an explicit discussion in an issue.

- **Observe only.** Never replace or modify the Amazon reader, and never
  write to the Kindle's own data files. The single exception is our own rows
  in `/var/local/appreg.db` (to register the Endpaper page), with a backup
  taken first.
- **No KOReader** or other replacement reader.
- **Lowest common denominator.** Code must work on older jailbroken Kindles
  and firmware, not only the newest. No per-model configuration unless
  there is no other way.
- **Simple install.** Unzip the release onto the Kindle, tap the library
  item, sign in with a code. No SSH, no terminal.
- **Never guess.** If a book cannot be matched to exactly one Hardcover book,
  skip it. Progress is forward only.
- **Endpaper never deletes anything on its own.** Only the Uninstall button
  removes files. (Some Kindles keep `/mnt/us` mounted over USB/MTP, so
  "folder deleted" heuristics are unsafe.)
- **Low drain.** Event-driven, no busy polling, minimal flash writes,
  no network use while offline.

## Design decisions

1. **Companion daemon, not a reader.** Triggers: LIPC events (left book,
   sleep, Wi-Fi up) and inotify on the Kindle's databases; a 15-minute poll
   is only a safety net. See `docs/architecture.md`.
2. **Data sources, in priority order:** the Kindle's `cc.db` → book files and
   sidecars → LIPC properties. LIPC is never required.
3. **Position = percent** from the first usable source: `cc.db`
   `p_percentFinished`, then `p_lastAccessedPosition` ÷ text length, then the
   `.azw3f` sidecar (fallback only; never finishes a book). Hardcover page =
   percent × edition pages. APNX page maps are not used.
4. **Capability model.** Every backend (database, sidecar, LIPC, inotify) is
   optional; a missing one must never stop the daemon.
5. **Auth:** Hardcover OAuth Device Authorization Grant (code/QR on the
   Kindle, approve on a phone). No token typing on the device.
6. **Runtime:** one static ARM binary in Go (`CGO_ENABLED=0`, `GOARM=6`) with
   bundled CA roots. Do not depend on system curl, Python or libraries.
   Builds use Go 1.23 (Linux 2.6.32+); newer Go needs Linux 3.2+, which
   rules out the oldest Kindles.
7. **UI:** a Mesquite (Kindle web app) page talking to the daemon over
   `127.0.0.1` with a random port and key; eips text screens as a fallback.
8. **State:** our own SQLite database (`modernc.org/sqlite`, WAL, diff
   writes) in `/var/local/endpaper`; JSON files only as a fallback.

## Repository layout

```
cmd/endpaper/        the program: launcher (library tap), daemon, local page
                     API, uninstall, developer commands
cmd/endpaper/ui/     the Mesquite page (ES5 HTML/JS, embedded in the binary)
internal/readers/    cc.db reader
internal/sidecar/    KRDS sidecar reader (.azw3f)
internal/mobi/       EXTH metadata (ISBN, ASIN) from book files
internal/book/       book identity and title/author rules
internal/match/      identify waterfall (ASIN → ISBN → library → search)
internal/syncer/     progress, finish, re-read, ratings, highlights
internal/daemon/     change detection, retries, state
internal/store/      the SQLite state store
internal/hardcover/  OAuth device flow, token refresh, GraphQL client
internal/metrics/    fmcache.db reader (end-of-book star ratings)
internal/kindledb/   retry for reads of the Kindle's databases mid-commit
internal/clippings/  My Clippings.txt parser
internal/events/     LIPC and inotify watchers
internal/appreg/     our rows in the Kindle's appreg.db
internal/screen/     eips text and pillow alert boxes
internal/config/     paths and token storage
internal/atomicfile/ crash-safe file writes
internal/certs/      bundled CA roots
packaging/kindle/    library item (Endpaper.sh) and its cover image
scripts/build.sh     builds the release zip
site/                landing page (GitHub Pages)
docs/                architecture, testing, decisions, research
```

## Build and test

```sh
go vet ./...
go test -race ./...
gofmt -l cmd internal          # must print nothing
HC_CLIENT_ID=<client id> ./scripts/build.sh   # → dist/endpaper-<version>.zip
```

- `cmd/endpaper/ui/app.js` must stay **ES5** (the Kindle's WebKit is old):
  no arrow functions, `let`/`const`, `fetch` or promises.
- Real device data for replay tests lives in `internal/*/testdata`.
- Device testing follows the per-build checklist in `docs/testing.md`.

## Releases

Publishing a GitHub release with a new `vX.Y.Z` tag runs
`.github/workflows/release.yml`: tests, an ARM build with the public OAuth
client ID (repository secret `HC_CLIENT_ID`), and the zip attached to the
release. Record user-facing changes in `CHANGELOG.md`.

## Conventions

- Mark anything not verified on a real device or in real documentation as
  **UNVERIFIED** (in code comments and docs).
- Keep changes small and testable; add a unit test for each bug fix.
- Prefer plain, simple technical English in docs and messages.
- Time spans that must count real time use the wall clock
  (`time.Now().Round(0)`): the monotonic clock stops while the Kindle sleeps.
- Never write progress, ratings or notes to a book unless the match is
  certain.

## Docs

- `docs/README.md` — index
- `docs/architecture.md` — parts, triggers, one scan, packages
- `docs/testing.md` — fixtures, self-test, device checklists
- `docs/roadmap.md` — goals, decisions and their history
- `docs/open-questions.md` — what is still unknown
- `docs/decisions/` — design decisions (state store, telemetry draft)
- `docs/research/` — device findings, Kindle data sources, KindleModding
  notes, Hardcover API and features, references
- `CHANGELOG.md` — user-facing changes per release
