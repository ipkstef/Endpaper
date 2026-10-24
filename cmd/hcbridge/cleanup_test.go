package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallGone(t *testing.T) {
	home := filepath.Join(t.TempDir(), "hcbridge")
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
