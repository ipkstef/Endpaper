package kindledb

import (
	"context"
	"errors"
	"testing"
)

func TestRetryBusy(t *testing.T) {
	n := 0
	v, err := RetryBusy(context.Background(), func() (int, error) {
		n++
		if n < 2 {
			return 0, errors.New("attempt to write a readonly database (776)")
		}
		return 7, nil
	})
	if v != 7 || err != nil || n != 2 {
		t.Fatalf("got %d %v after %d tries", v, err, n)
	}
	n = 0
	_, err = RetryBusy(context.Background(), func() (int, error) { n++; return 0, errors.New("no such table") })
	if err == nil || n != 1 {
		t.Fatalf("other errors must not retry: %v, %d tries", err, n)
	}
}
