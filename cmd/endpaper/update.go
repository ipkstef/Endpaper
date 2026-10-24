package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// latestURL is a variable for tests.
var latestURL = "https://api.github.com/repos/ipkstef/endpaper/releases/latest"

const (
	// ReleasesURL is shown on the page when an update is available.
	releasesURL = "github.com/ipkstef/endpaper/releases"
	// updateEvery: look for a new release at most this often, and only
	// while the page is open (no background network use for this). A
	// failed check is tried again after updateRetry.
	updateEvery = time.Hour
	updateRetry = 5 * time.Minute
)

// updates remembers the newest release (in memory only: no DB write).
type updates struct {
	http *http.Client

	mu     sync.Mutex
	next   time.Time // no check before this (wall clock)
	busy   bool
	latest string
	logged string // last result written to the log
}

// available returns the newer release tag ("" if none) and whether a check
// is running (the page then asks again soon). It starts a background check
// when the last one is older than updateEvery.
func (u *updates) available(current string) (tag string, checking bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if !u.busy && !time.Now().Before(u.next) && released(current) {
		u.busy = true
		go u.check(current)
	}
	if newer(u.latest, current) {
		tag = u.latest
	}
	return tag, u.busy
}

func (u *updates) check(current string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	tag, err := latestTag(ctx, u.http)
	u.mu.Lock()
	defer u.mu.Unlock()
	u.busy = false
	now := time.Now().Round(0) // wall clock: the Kindle sleeps
	msg := ""
	if err != nil {
		u.next = now.Add(updateRetry)
		msg = "update check failed: " + err.Error()
	} else {
		u.next = now.Add(updateEvery)
		u.latest = tag
		msg = fmt.Sprintf("update check: latest %s, running %s", tag, current)
	}
	if msg != u.logged { // one line per new result, not one per hour
		log.Print(msg)
		u.logged = msg
	}
}

func latestTag(ctx context.Context, c *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Endpaper")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github: %s", resp.Status)
	}
	var r struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.Tag, nil
}

// semver parses "v1.2.3" (also "v1.2.3-4-gabc", a build after the tag).
func semver(v string) ([3]int, bool) {
	var out [3]int
	if !strings.HasPrefix(v, "v") {
		return out, false
	}
	v = strings.TrimPrefix(v, "v")
	if i := strings.IndexByte(v, '-'); i >= 0 {
		v = v[:i]
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// released reports if this build is a release (development builds never
// look for updates).
func released(current string) bool {
	_, ok := semver(current)
	return ok
}

// newer reports if latest is a higher version than current.
func newer(latest, current string) bool {
	l, ok1 := semver(latest)
	c, ok2 := semver(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := range l {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}
