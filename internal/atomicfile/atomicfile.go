// Package atomicfile writes files so that a crash or power loss leaves either
// the old or the new content, never an empty or half file.
package atomicfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// MinFree: state files are not written when the folder's file system would
// have less free space than this. /var/local is shared with the Kindle
// system (cc.db); filling it could break the Kindle.
var MinFree int64 = 5 << 20

// ErrLowSpace means a write was skipped to keep MinFree free.
var ErrLowSpace = errors.New("low disk space, not saved")

// Guarded is Write with the free-space check (for state that can be rebuilt
// or re-sent). Use Write for small files that must never be lost (token).
func Guarded(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	// The temp file and the old file exist together for a moment.
	if err := CheckFree(dir, int64(len(data))); err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return Write(path, data, perm)
}

// CheckFree returns ErrLowSpace when writing need more bytes in dir would
// leave less than MinFree. If the space cannot be read, it allows the write.
func CheckFree(dir string, need int64) error {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return nil
	}
	if free := int64(st.Bavail) * int64(st.Bsize); free-need < MinFree {
		return fmt.Errorf("%w (%d KB free)", ErrLowSpace, free>>10)
	}
	return nil
}

// CleanTemp removes temp files that Write left in dir after a crash or
// power loss (older than an hour, so a write running now is not touched).
// It returns how many it removed.
func CleanTemp(dir string) int {
	paths, _ := filepath.Glob(filepath.Join(dir, "*.tmp*"))
	n := 0
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() &&
			time.Since(fi.ModTime()) > time.Hour && os.Remove(p) == nil {
			n++
		}
	}
	return n
}

// GuardedJSON writes v as indented JSON with Guarded.
func GuardedJSON(path string, v any, perm os.FileMode) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return Guarded(path, b, perm)
}

// Write writes data to path: a unique temp file in the same folder, fsync,
// rename, then fsync of the folder. Mode is perm (the folder is made 0700).
func Write(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()
	if err := f.Chmod(perm); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	ok = true
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}

// WriteJSON writes v as indented JSON with Write.
func WriteJSON(path string, v any, perm os.FileMode) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return Write(path, b, perm)
}
