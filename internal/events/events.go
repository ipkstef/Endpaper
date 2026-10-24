// Package events watches Kindle events: LIPC (lipc-wait-event) and inotify.
// Findings on FW 5.17.1: docs/research/findings-bellatrix-5.17.1.md, "Events test".
package events

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// LIPC runs `lipc-wait-event -m <publisher> '*'` and calls onLine for each
// event line. It restarts the tool if it exits. It returns an error at once
// if lipc-wait-event is missing; otherwise it runs until ctx is done.
func LIPC(ctx context.Context, publisher string, onLine func(string), logf func(string, ...any)) error {
	bin, err := exec.LookPath("lipc-wait-event")
	if err != nil {
		return err
	}
	go func() {
		backoff := time.Second
		for ctx.Err() == nil {
			start := time.Now()
			cmd := exec.CommandContext(ctx, bin, "-m", publisher, "*")
			// The child dies with the daemon (else it stays until its next
			// event).
			cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
			out, err := cmd.StdoutPipe()
			if err == nil {
				err = cmd.Start()
			}
			if err == nil {
				sc := bufio.NewScanner(out)
				sc.Buffer(make([]byte, 4096), 1<<20)
				for sc.Scan() {
					onLine(sc.Text())
				}
				if serr := sc.Err(); serr != nil {
					// Nobody reads the pipe any more: stop the child, or
					// Wait blocks for good once the pipe is full.
					_ = cmd.Process.Kill()
					logf("events: lipc %s: %v", publisher, serr)
				}
				err = cmd.Wait()
			}
			if ctx.Err() != nil {
				return
			}
			if time.Since(start) > time.Minute {
				backoff = time.Second
			}
			logf("events: lipc %s exited (%v), restart in %s", publisher, err, backoff)
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < time.Minute {
				backoff *= 2
			}
		}
	}()
	return nil
}

// Inotify masks we use.
const (
	InModify     = syscall.IN_MODIFY
	InCloseWrite = syscall.IN_CLOSE_WRITE
	InCreate     = syscall.IN_CREATE
	InDelete     = syscall.IN_DELETE
	InMovedTo    = syscall.IN_MOVED_TO
)

// WatchDir watches a directory with inotify and calls onEvent for each event.
// It returns an error at once if inotify is not available. The returned
// channel is closed when the watch ends: ctx done, or the folder is gone or
// unmounted (e.g. /mnt/us in USB mode); the caller may then watch again.
func WatchDir(ctx context.Context, dir string, onEvent func(name string, mask uint32)) (<-chan struct{}, error) {
	fd, err := initWatch(dir)
	if err != nil {
		return nil, err
	}
	// Non-blocking fd in an *os.File: reads go through Go's poller, so
	// Close wakes a waiting Read (a raw blocking read(2) would not wake,
	// and would later read from a reused fd number).
	f := os.NewFile(uintptr(fd), "inotify:"+dir)
	done := make(chan struct{})
	stop := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
		case <-stop:
		}
		f.Close()
	}()
	go func() {
		defer close(done)
		defer close(stop)
		buf := make([]byte, 64*1024)
		for {
			n, err := f.Read(buf)
			if err != nil {
				if errors.Is(err, syscall.EINTR) {
					continue
				}
				return
			}
			if n <= 0 {
				return
			}
			for off := 0; off+syscall.SizeofInotifyEvent <= n; {
				ev := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[off]))
				end := off + syscall.SizeofInotifyEvent + int(ev.Len)
				if end > n {
					break
				}
				name := strings.TrimRight(string(buf[off+syscall.SizeofInotifyEvent:end]), "\x00")
				if ev.Mask&(syscall.IN_IGNORED|syscall.IN_UNMOUNT) != 0 {
					return // watch removed: folder deleted or unmounted
				}
				onEvent(name, ev.Mask)
				off = end
			}
		}
	}()
	return done, nil
}

const watchMask = uint32(InModify | InCloseWrite | InCreate | InDelete | InMovedTo)

// initWatch makes a non-blocking inotify fd watching dir. CLOEXEC: child
// processes (lipc-wait-event, lipc-get-prop) must not inherit it.
func initWatch(dir string) (int, error) {
	fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
	if err != nil {
		return -1, err
	}
	if _, err := syscall.InotifyAddWatch(fd, dir, watchMask); err != nil {
		syscall.Close(fd)
		return -1, err
	}
	return fd, nil
}

// CanWatch reports if dir can be watched (for the self-test): it adds a
// watch and closes it at once, with no reader left behind.
func CanWatch(dir string) error {
	fd, err := initWatch(dir)
	if err != nil {
		return err
	}
	return syscall.Close(fd)
}
