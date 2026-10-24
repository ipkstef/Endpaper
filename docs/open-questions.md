# Open questions and risks

Answered questions and how they were answered: `docs/roadmap.md` and
`docs/research/findings-bellatrix-5.17.1.md`.

## Device coverage
- [ ] **Kernel floor.** Go 1.24+ needs Linux 3.2+, Go 1.23 needs 2.6.32+;
      builds use Go 1.23. Kindle kernels (**UNVERIFIED**): Touch / PW1 about
      2.6.31, PW2 / PW3 / Voyage about 3.0.35, newer 4.x. Needs a run on an
      old device; then pin the Go version.
- [ ] Other models and firmware: `cc.db` columns, sidecar format, LIPC
      events, Mesquite and `appreg.db` are only tested on FW 5.17.1.
- [ ] Older jailbreaks without scriptlet support (no SH_Integration):
      no install path today.
- [ ] Library cover from the scriptlet `# Icon:` header (**UNVERIFIED**).

## Product
- [ ] Shelf choice on the end-of-book screen → Hardcover status: built,
      off in the first release (`syncShelves`).
- [ ] Opt-in telemetry: draft only, out of scope (`docs/decisions/telemetry.md`).
- [ ] Autostart at boot: needs a rootfs write; not planned.

## Known limits
- Books with poor metadata (no ISBN/ASIN, odd titles) are not matched;
  the user adds the book on hardcover.app. Endpaper never guesses.
- Progress is forward only by design.
