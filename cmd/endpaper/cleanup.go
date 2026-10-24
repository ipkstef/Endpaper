package main

import (
	"log"
	"os"
	"strings"

	"github.com/ipkstef/endpaper/internal/appreg"
)

const (
	usbRoot   = "/mnt/us"
	scriptlet = "/mnt/us/documents/Endpaper.sh"
)

// uninstall removes everything we put on the Kindle, then exits: our
// appreg rows, the page, the state (token, DB, log), the scriptlet, the
// install folder and the running binary in /tmp (Linux lets a running
// program delete its own file). Only the page's Uninstall button calls it:
// Endpaper never deletes anything on its own (user decision 2026-10-24).
func (a *app) uninstall() {
	log.Printf("uninstall: removing endpaper")
	if err := appreg.Unregister(appreg.DefaultPath, appID); err != nil {
		log.Printf("uninstall: appreg: %v", err)
	}
	_ = os.RemoveAll(uiDir)
	_ = os.Remove(scriptlet)
	_ = os.RemoveAll(scriptlet + ".sdr")
	if a.home != "" {
		_ = os.RemoveAll(a.home)
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
