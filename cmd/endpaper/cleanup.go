package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ipkstef/endpaper/internal/appreg"
)

const (
	usbRoot   = "/mnt/us"
	scriptlet = "/mnt/us/documents/Endpaper.sh"
)

// usbMounted reports if the USB drive (/mnt/us) is mounted and readable.
// In USB mode the Kindle unmounts it, so every file there looks deleted.
func usbMounted() bool {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return false
	}
	defer f.Close()
	found := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if fs := strings.Fields(sc.Text()); len(fs) > 1 && fs[1] == usbRoot {
			found = true
			break
		}
	}
	if !found {
		return false
	}
	_, err = os.Stat(filepath.Join(usbRoot, "documents"))
	return err == nil
}

// installGone reports if the user deleted the install folder. It is never
// true while the drive is unmounted (USB mode), or when home is not set
// (development runs).
func installGone(home string, mounted func() bool) bool {
	if home == "" || !mounted() {
		return false
	}
	_, err := os.Stat(home)
	return os.IsNotExist(err)
}

// uninstall removes everything we put on the Kindle, then exits:
// our appreg rows, the page, the state (token, DB, log), the scriptlet,
// the install folder (when removeHome) and the running binary in /tmp
// (Linux lets a running program delete its own file).
func (a *app) uninstall(removeHome bool) {
	log.Printf("uninstall: removing endpaper (folder removed by user: %v)", !removeHome)
	if err := appreg.Unregister(appreg.DefaultPath, appID); err != nil {
		log.Printf("uninstall: appreg: %v", err)
	}
	_ = os.RemoveAll(uiDir)
	if usbMounted() {
		_ = os.Remove(scriptlet)
		_ = os.RemoveAll(scriptlet + ".sdr")
		if removeHome && a.home != "" {
			_ = os.RemoveAll(a.home)
		}
	}
	if a.sdb != nil {
		a.sdb.Close()
		a.sdb = nil
	}
	_ = os.RemoveAll(a.stateDir)
	if exe, err := os.Executable(); err == nil && strings.HasPrefix(exe, "/tmp/") {
		_ = os.Remove(exe)
	}
	os.Exit(0)
}

// legacy: the files of the earlier name "hcbridge" (KUAL builds and the
// first scriptlet build). Variables, so tests can point them elsewhere.
var legacy = struct {
	stateDir, uiDir, appID string
	usbPaths               []string // removed when /mnt/us is mounted
}{
	stateDir: "/var/local/hcbridge",
	uiDir:    "/var/local/mesquite/hcbridge",
	appID:    "io.github.ipkstef.hcbridge",
	usbPaths: []string{
		"/mnt/us/documents/Hardcover.sh",
		"/mnt/us/documents/Hardcover.sh.sdr",
		"/mnt/us/hcbridge",
		"/mnt/us/extensions/hcbridge",
	},
}

// legacyNames maps old file names in the state folder to new ones.
var legacyNames = map[string]string{
	"hcbridge.db":     "endpaper.db",
	"hcbridge.db-wal": "endpaper.db-wal",
	"hcbridge.db-shm": "endpaper.db-shm",
	"hcbridge.log":    "endpaper.log",
	"hcbridge.log.1":  "endpaper.log.1",
}

// migrateLegacy moves an install of the old name to Endpaper, once: it
// stops the old daemon, moves the state (sign-in, DB, log) to stateDir,
// and removes the old page, appreg rows and old files on the USB drive.
// It runs before the log is set up (the log lives in stateDir), so it
// returns its notes for the log.
func migrateLegacy(stateDir string, mounted func() bool) (notes []string) {
	if _, err := os.Stat(legacy.stateDir); err != nil {
		return nil
	}
	if pid := lockedPid(filepath.Join(legacy.stateDir, "daemon.pid")); pid != 0 {
		_ = syscall.Kill(pid, syscall.SIGTERM)
		for i := 0; i < 40 && lockedPid(filepath.Join(legacy.stateDir, "daemon.pid")) != 0; i++ {
			time.Sleep(250 * time.Millisecond)
		}
		notes = append(notes, fmt.Sprintf("stopped old daemon (pid %d)", pid))
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return append(notes, "migrate: "+err.Error())
	}
	entries, _ := os.ReadDir(legacy.stateDir)
	moved := 0
	for _, e := range entries {
		name := e.Name()
		if n, ok := legacyNames[name]; ok {
			name = n
		}
		dst := filepath.Join(stateDir, name)
		if _, err := os.Stat(dst); err == nil {
			continue // never overwrite newer state
		}
		if os.Rename(filepath.Join(legacy.stateDir, e.Name()), dst) == nil {
			moved++
		}
	}
	notes = append(notes, fmt.Sprintf("moved %d files from %s", moved, legacy.stateDir))
	_ = os.RemoveAll(legacy.stateDir)
	if err := appreg.Unregister(appreg.DefaultPath, legacy.appID); err == nil {
		notes = append(notes, "removed old appreg rows")
	}
	_ = os.RemoveAll(legacy.uiDir)
	if mounted() {
		for _, p := range legacy.usbPaths {
			if _, err := os.Stat(p); err == nil && os.RemoveAll(p) == nil {
				notes = append(notes, "removed "+p)
			}
		}
	}
	return notes
}
