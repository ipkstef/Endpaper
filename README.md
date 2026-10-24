<h1 align="center"><img src="docs/img/logo.png" width="260" alt="Endpaper"></h1>

<p align="center"><b>A native Kindle → <a href="https://hardcover.app">Hardcover.app</a> sync.</b></p>

<p align="center">
  <a href="https://github.com/ipkstef/Endpaper/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/ipkstef/Endpaper?include_prereleases&color=161616"></a>
  <a href="https://github.com/ipkstef/Endpaper/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/ipkstef/Endpaper/actions/workflows/ci.yml/badge.svg"></a>
  <a href="LICENSE"><img alt="MIT license" src="https://img.shields.io/badge/license-MIT-161616"></a>
</p>

<p align="center"><a href="https://ipkstef.github.io/Endpaper/">Website</a> ·
<a href="https://github.com/ipkstef/Endpaper/releases/latest">Download</a> ·
<a href="CHANGELOG.md">Changelog</a></p>

Read on your jailbroken Kindle as usual. Endpaper is a small background
program that sends your reading progress to Hardcover.

- **Progress:** the page you reached, after you close the book or the
  Kindle goes to sleep. Forward only: it never moves your progress back.
- **New books:** a book you start reading is added to *Currently Reading*.
- **Finished books:** marked *Read* at the end of the book.
- **Re-reads:** starting a finished book again starts a new read.
- **Star ratings** from the Kindle's end-of-book screen.
- **Highlights and notes:** sent as *private* journal entries.

Endpaper is an independent project, not made by or affiliated with
Hardcover or Amazon.

It only reads the Kindle's files. It does not change the Kindle reader or
your books, and it works with the stock Amazon reader (no KOReader).

## Requirements

- A jailbroken Kindle that can run *scriptlets* (`.sh` files that show up in
  the library). Current jailbreaks can do this.
- A Hardcover account.
- Wi-Fi for syncing. Without Wi-Fi, changes wait and are sent later.

Tested on one Kindle with firmware 5.17.1. Other models and firmware
should work but are not tested yet; please report results.

## Install

**Before you start (recommended):** unlink Goodreads on the Kindle
(Settings > Your Account; the exact menu name varies by firmware). Endpaper works
either way, but with Goodreads linked the Kindle shows a Goodreads
"Rating Error" box after a star rating on sideloaded books, and it may also
send your progress to Goodreads. Your star taps still reach Hardcover when
Goodreads is unlinked.

1. Download `endpaper-<version>.zip` from the
   [releases page](https://github.com/ipkstef/endpaper/releases).
2. Connect the Kindle to your computer with USB.
3. Unzip the file onto the Kindle drive. You get:
   - `documents/Endpaper.sh`: the **Endpaper** item in your library
   - `endpaper/`: the program
4. Eject the Kindle.
5. In the library, tap **Endpaper**. Tap **Sign in**, scan the code with
   your phone (or open the link and enter the code), and approve.

Sync now runs in the background. You do not need to open Endpaper again.

**After a restart of the Kindle**, tap **Endpaper** once to start sync
again.

## Use

Tap **Endpaper** in the library to see the status and these options:

| Button | What it does |
|---|---|
| Sync now | Checks your books at once (normally automatic) |
| Import old highlights | Sends highlights and notes made before you installed |
| Check setup | Tests each part and shows OK / FAIL |
| Save log for support | Writes `endpaper/endpaper-log.txt` to the USB drive |
| Sign out | Removes your sign-in from this Kindle |
| Stop sync | Stops until you tap Endpaper again |
| Uninstall | Removes everything (see below) |

Leave the page with the Home button.

### Tips

- A book that is not found on Hardcover is skipped. Add the right edition to
  any shelf on hardcover.app; it syncs within an hour.

## Update

When a new version is out, the Endpaper page shows an **Update available**
banner.

1. Download the new zip from the releases page.
2. Connect USB and unzip it onto the Kindle drive (replace the old files,
   or delete them first: either is fine).
3. Eject and tap **Endpaper**. The new version starts; you stay signed in.

## Uninstall

Tap **Endpaper** > **Uninstall**. It removes the app, its settings and
your sign-in from the Kindle.

Endpaper never deletes anything on its own. If you only delete
`documents/Endpaper.sh` and the `endpaper` folder over USB, sync stops at
the next restart, but your sign-in and settings stay in
`/var/local/endpaper` (a few KB). To remove them too, install again and
use **Uninstall**.

## Privacy

Your Hardcover sign-in stays on the Kindle (`/var/local/endpaper`). The
program talks to Hardcover, and, while the Endpaper page is open, at most
once an hour to GitHub to look for a new version. Highlights and notes are
sent as private entries. Nothing else is collected.

## Troubleshooting

1. Tap **Endpaper** > **Check setup**.
2. Tap **Save log for support**, connect USB and send
   `endpaper/endpaper-log.txt` with your report.

## For developers

```sh
go test ./...
HC_CLIENT_ID=<your Hardcover OAuth client id> ./scripts/build.sh   # dist/endpaper-<version>.zip
```

One static ARM binary in Go (no system libraries needed). How it works:
[`docs/architecture.md`](docs/architecture.md); all docs:
[`docs/`](docs/README.md).

```
cmd/endpaper/        the program: launcher, daemon, local page API
cmd/endpaper/ui/     the Kindle page (embedded in the binary)
internal/            readers, sync logic, Hardcover client, state DB, ...
packaging/kindle/    library item (Endpaper.sh) and its cover
scripts/build.sh     builds dist/endpaper-<version>.zip
site/                the website (GitHub Pages)
docs/                architecture, testing, decisions, research
```

Releases: set the repository secret `HC_CLIENT_ID`, then push a tag such
as `v0.1.0`. The release workflow tests, builds and publishes the zip.

## License

MIT, see `LICENSE`.
