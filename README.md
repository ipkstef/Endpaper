# Hardcover Sync for Kindle

Read on your jailbroken Kindle as usual. This small background program
sends your reading progress to [Hardcover](https://hardcover.app).

- **Progress:** the page you reached, after you close the book or the
  Kindle goes to sleep. Forward only: it never moves your progress back.
- **New books:** a book you start reading is added to *Currently Reading*.
- **Finished books:** marked *Read* at the end of the book.
- **Re-reads:** starting a finished book again starts a new read.
- **Star ratings** from the Kindle's end-of-book screen.
- **Highlights and notes:** sent as *private* journal entries.

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

1. Download `hcbridge-<version>.zip` from the
   [releases page](https://github.com/ipkstef/kindle-hardcover-bridge/releases).
2. Connect the Kindle to your computer with USB.
3. Unzip the file onto the Kindle drive. You get:
   - `documents/Hardcover.sh`: the **Hardcover** item in your library
   - `hcbridge/`: the program
4. Eject the Kindle.
5. In the library, tap **Hardcover**. Tap **Sign in**, scan the code with
   your phone (or open the link and enter the code), and approve.

Sync now runs in the background. You do not need to open Hardcover again.

**After a restart of the Kindle**, tap **Hardcover** once to start sync
again.

## Use

Tap **Hardcover** in the library to see the status and these options:

| Button | What it does |
|---|---|
| Sync now | Checks your books at once (normally automatic) |
| Import old highlights | Sends highlights and notes made before you installed |
| Check setup | Tests each part and shows OK / FAIL |
| Save log for support | Writes `hcbridge/hcbridge-log.txt` to the USB drive |
| Sign out | Removes your sign-in from this Kindle |
| Stop sync | Stops until you tap Hardcover again |
| Uninstall | Removes everything (see below) |

Leave the page with the Home button.

### Tips

- **Unlink Goodreads** on the Kindle (Settings > Your Account) to stop
  Goodreads error boxes after a rating. Ratings still go to Hardcover.
- A book that is not found on Hardcover is skipped. Add the right edition to
  any shelf on hardcover.app; it syncs within an hour.

## Uninstall

Either:

- tap **Hardcover** > **Uninstall**, or
- delete `documents/Hardcover.sh` and the `hcbridge` folder over USB. If
  sync is running, it removes its remaining files (sign-in and state) the
  next time it wakes up. If you restarted the Kindle and did not tap
  Hardcover since then, tap it once before you delete the folder.

## Privacy

Your Hardcover sign-in stays on the Kindle (`/var/local/hcbridge`). The
program only talks to Hardcover. Highlights and notes are sent as private
entries. Nothing else is collected.

## Troubleshooting

1. Tap **Hardcover** > **Check setup**.
2. Tap **Save log for support**, connect USB and send
   `hcbridge/hcbridge-log.txt` with your report.

## For developers

```sh
go test ./...
HC_CLIENT_ID=<your Hardcover OAuth client id> ./scripts/build.sh   # dist/hcbridge-<version>.zip
```

One static ARM binary in Go (no system libraries needed). The design and
the research behind it are in `docs/` (start with `docs/architecture.md`).
