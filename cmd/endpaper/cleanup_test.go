package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallGone(t *testing.T) {
	home := filepath.Join(t.TempDir(), "endpaper")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}
	up := func() bool { return true }
	down := func() bool { return false }
	if installGone(home, up) {
		t.Fatal("folder there: not gone")
	}
	os.Remove(home)
	if !installGone(home, up) {
		t.Fatal("folder deleted: gone")
	}
	// USB mode: /mnt/us unmounted, the folder only looks deleted.
	if installGone(home, down) {
		t.Fatal("USB mode must never count as deleted")
	}
	if installGone("", up) {
		t.Fatal("no home (development run): never gone")
	}
}

func TestMigrateLegacy(t *testing.T) {
	old := filepath.Join(t.TempDir(), "hcbridge")
	usbOld := filepath.Join(t.TempDir(), "usb-hcbridge")
	saved := legacy
	defer func() { legacy = saved }()
	legacy.stateDir, legacy.uiDir, legacy.usbPaths = old, filepath.Join(t.TempDir(), "ui"), []string{usbOld}
	legacy.appID = "test.none" // appreg.db is not there: ignored

	newDir := filepath.Join(t.TempDir(), "endpaper")
	if n := migrateLegacy(newDir, func() bool { return true }); n != nil {
		t.Fatalf("nothing to move: %v", n)
	}
	os.MkdirAll(old, 0o700)
	os.MkdirAll(usbOld, 0o755)
	os.WriteFile(filepath.Join(old, "token.json"), []byte(`{"access_token":"x"}`), 0o600)
	os.WriteFile(filepath.Join(old, "hcbridge.db"), []byte("db"), 0o600)
	// The new log may exist already (setup order): never overwritten.
	os.MkdirAll(newDir, 0o700)
	os.WriteFile(filepath.Join(newDir, "endpaper.log"), []byte("new"), 0o600)
	os.WriteFile(filepath.Join(old, "hcbridge.log"), []byte("old"), 0o600)

	notes := migrateLegacy(newDir, func() bool { return true })
	if len(notes) == 0 {
		t.Fatal("no notes")
	}
	for _, f := range []string{"token.json", "endpaper.db"} {
		if _, err := os.Stat(filepath.Join(newDir, f)); err != nil {
			t.Fatalf("%s not moved", f)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(newDir, "endpaper.log")); string(b) != "new" {
		t.Fatal("new log overwritten")
	}
	for _, p := range []string{old, usbOld} {
		if _, err := os.Stat(p); err == nil {
			t.Fatalf("%s not removed", p)
		}
	}
}

// An old WAL is never put next to a newer Endpaper database.
func TestMigrateKeepsDBFilesTogether(t *testing.T) {
	old := filepath.Join(t.TempDir(), "hcbridge")
	saved := legacy
	defer func() { legacy = saved }()
	legacy.stateDir, legacy.uiDir, legacy.usbPaths, legacy.appID = old, t.TempDir(), nil, "test.none"
	newDir := filepath.Join(t.TempDir(), "endpaper")
	os.MkdirAll(old, 0o700)
	os.MkdirAll(newDir, 0o700)
	os.WriteFile(filepath.Join(old, "hcbridge.db"), []byte("old"), 0o600)
	os.WriteFile(filepath.Join(old, "hcbridge.db-wal"), []byte("old wal"), 0o600)
	os.WriteFile(filepath.Join(newDir, "endpaper.db"), []byte("new"), 0o600)
	migrateLegacy(newDir, func() bool { return false })
	if _, err := os.Stat(filepath.Join(newDir, "endpaper.db-wal")); err == nil {
		t.Fatal("old WAL moved next to the new database")
	}
	if b, _ := os.ReadFile(filepath.Join(newDir, "endpaper.db")); string(b) != "new" {
		t.Fatal("new database changed")
	}
}
