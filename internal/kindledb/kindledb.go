// Package kindledb has helpers for reading the Kindle's own SQLite files
// (cc.db, fmcache.db) read-only while the Kindle may be writing them.
package kindledb

import (
	"context"
	"strings"
	"time"
)

// RetryBusy runs read again (up to 3 times, 1 s apart) when the Kindle is in
// the middle of a commit: a hot journal gives "readonly database (776)" on a
// read-only open, a lock gives "database is locked" (device log 2026-09-28).
func RetryBusy[T any](ctx context.Context, read func() (T, error)) (T, error) {
	for i := 0; ; i++ {
		v, err := read()
		if err == nil || i >= 3 || !Busy(err) {
			return v, err
		}
		select {
		case <-ctx.Done():
			return v, err
		case <-time.After(time.Second):
		}
	}
}

// Busy reports an error that goes away when the Kindle's commit ends.
func Busy(err error) bool {
	s := err.Error()
	return strings.Contains(s, "readonly database") || strings.Contains(s, "database is locked") ||
		strings.Contains(s, "SQLITE_BUSY")
}
