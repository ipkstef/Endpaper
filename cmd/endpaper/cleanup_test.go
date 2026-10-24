package main

import (
	"os"
	"path/filepath"
	"testing"
)

// Only Mesquite running our app id is found (cmdline is NUL-separated).
func TestPagePids(t *testing.T) {
	proc := t.TempDir()
	write := func(pid, cmd string) {
		os.MkdirAll(filepath.Join(proc, pid), 0o755)
		os.WriteFile(filepath.Join(proc, pid, "cmdline"), []byte(cmd), 0o644)
	}
	write("100", "/usr/bin/mesquite\x00-l\x00"+appID+"\x00-c\x00file:///var/local/mesquite/endpaper/\x00")
	write("101", "/usr/bin/mesquite\x00-l\x00com.lab126.store\x00-c\x00file:///x/\x00")
	write("102", "/tmp/endpaper\x00daemon\x00-l\x00"+appID+"\x00")
	write("self", "x")
	got := pagePids(proc)
	if len(got) != 1 || got[0] != 100 {
		t.Fatalf("got %v, want [100]", got)
	}
}
