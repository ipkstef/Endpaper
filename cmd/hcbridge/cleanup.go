package main

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipkstef/kindle-hardcover-bridge/internal/appreg"
)

const (
	usbRoot   = "/mnt/us"
	scriptlet = "/mnt/us/documents/Hardcover.sh"
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
	log.Printf("uninstall: removing hcbridge (folder removed by user: %v)", !removeHome)
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
