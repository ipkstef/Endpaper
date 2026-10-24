# Changelog

## Unreleased (v0.1.2)
- **Fix:** updating over USB could uninstall Endpaper and sign you out
  (Kindles where the drive stays mounted during USB). Endpaper now never
  deletes anything on its own; only the **Uninstall** button removes it.
- Each tap on Endpaper opens a fresh page (an old page that was still
  open is closed first).

## v0.1.1 — 2026-10-24
- "Sync now" shows when the check is done ("Done. …"); old messages clear.
- Endpaper logo on the page, in the README and as the library cover.
- Landing page: https://ipkstef.github.io/Endpaper/
- Install guide: unlink Goodreads first (recommended).

Robustness (code review 2026-10-24):
- A book is never marked Read from a position that is past the end of
  the text; an ISBN found in a book description needs the same title.
- A failed lookup step means "try again", not "not found".
- Updates: the old background sync is always replaced (killed if stuck).
- The page reconnects after background sync restarted.
- Token: refreshed once when Hardcover refuses it (clock behind); a
  refreshed token is never lost when the disk is full; sign-out cannot be
  undone by a refresh at the same moment.
- Captive Wi-Fi portal pages count as "offline, try later".
- No duplicate highlights or lost star taps after a state read error or
  a run without the database; star taps in the "future" (wrong clock)
  are no longer skipped.
- cc.db reads retry while the Kindle commits (a sleep sync is not lost).
- Less battery use while offline with the page open; no leaked file
  watchers from "Check setup"; tools on the Kindle have time limits.
- The state database refuses writes when the disk is nearly full; temp
  files left by a power loss are removed.

## v0.1.0 — 2026-10-24
First release.
- Syncs reading progress (forward only), new books, finished books,
  re-reads, star ratings, and highlights/notes (private journal entries)
  from the stock Kindle reader to Hardcover.
- One library item, **Endpaper**, opens a page: sign in with a QR code,
  status, sync now, import old highlights, check setup, save log, sign out,
  stop, uninstall.
- Update banner when a newer release is out.
- Uninstall from the page, or by deleting the `endpaper` folder (removed
  in v0.1.2: unsafe on some Kindles).
- Moves an earlier `hcbridge` test install on first start.
