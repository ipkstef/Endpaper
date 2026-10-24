package events

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchDirEventAndStop(t *testing.T) {
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	got := make(chan string, 4)
	done, err := WatchDir(ctx, dir, func(name string, mask uint32) { got <- name })
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(dir, "x"), []byte("1"), 0o600)
	select {
	case n := <-got:
		if n != "x" {
			t.Fatalf("event for %q", n)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
	// Cancel must end the reader at once (no write needed to wake it).
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("reader did not stop after cancel")
	}
}

func TestCanWatch(t *testing.T) {
	if err := CanWatch(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	if err := CanWatch(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing folder: expected error")
	}
}
