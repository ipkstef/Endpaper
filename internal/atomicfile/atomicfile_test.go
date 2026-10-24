package atomicfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "a.json")
	if err := WriteJSON(p, map[string]int{"a": 1}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(p, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "new" {
		t.Fatalf("got %q, %v", b, err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", fi.Mode())
	}
	left, _ := filepath.Glob(filepath.Join(filepath.Dir(p), "*.tmp*"))
	if len(left) != 0 {
		t.Errorf("temp files left: %v", left)
	}
}

func TestGuardedLowSpace(t *testing.T) {
	old := MinFree
	defer func() { MinFree = old }()
	MinFree = 1 << 62 // more than any disk
	p := filepath.Join(t.TempDir(), "s.json")
	if err := GuardedJSON(p, 1, 0o600); !errors.Is(err, ErrLowSpace) {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("file written")
	}
	MinFree = 0
	if err := GuardedJSON(p, 1, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCleanTemp(t *testing.T) {
	dir := t.TempDir()
	oldTmp := filepath.Join(dir, "state.json.tmp123")
	newTmp := filepath.Join(dir, "state.json.tmp456")
	keep := filepath.Join(dir, "state.json")
	for _, p := range []string{oldTmp, newTmp, keep} {
		os.WriteFile(p, []byte("x"), 0o600)
	}
	past := time.Now().Add(-2 * time.Hour)
	os.Chtimes(oldTmp, past, past)
	os.Chtimes(keep, past, past)
	if n := CleanTemp(dir); n != 1 {
		t.Fatalf("removed %d, want 1", n)
	}
	for p, want := range map[string]bool{oldTmp: false, newTmp: true, keep: true} {
		if _, err := os.Stat(p); (err == nil) != want {
			t.Fatalf("%s: exists %v, want %v", filepath.Base(p), err == nil, want)
		}
	}
}
