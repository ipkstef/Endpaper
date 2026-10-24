package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/ipkstef/endpaper/internal/config"
	"github.com/ipkstef/endpaper/internal/hardcover"
)

func TestAPI(t *testing.T) {
	dir := t.TempDir()
	a := &app{store: config.NewTokenStore(dir), stateDir: dir}
	a.oauth = hardcover.NewOAuth(http.DefaultClient, "")
	fired := ""
	s := &api{a: a, key: newKey(), fire: func(w string) { fired = w }, stop: func() {}, importAll: new(atomic.Bool)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	port, err := s.serve(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if a.sdb != nil {
			a.sdb.Close()
		}
	}()
	get := func(method, path, key string) (int, map[string]any) {
		req, _ := http.NewRequest(method, fmt.Sprintf("http://127.0.0.1:%d/api/%s?k=%s", port, path, key), nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	if code, _ := get("GET", "status", "wrong"); code != http.StatusForbidden {
		t.Fatalf("bad key: %d", code)
	}
	code, out := get("GET", "status", s.key)
	if code != 200 || out["signedIn"] != false || out["version"] != version {
		t.Fatalf("status: %d %v", code, out)
	}
	// Not signed in: no sync, no sign-in without a client ID.
	if code, _ := get("POST", "sync", s.key); code != 500 || fired != "" {
		t.Fatalf("sync while signed out: %d %q", code, fired)
	}
	if code, out := get("POST", "login", s.key); code != 500 || out["error"] == nil {
		t.Fatalf("login without client ID: %d %v", code, out)
	}
	// Signed in: sync fires a scan.
	if err := a.store.Save(&hardcover.Token{AccessToken: "x"}); err != nil {
		t.Fatal(err)
	}
	if code, _ := get("POST", "sync", s.key); code != 200 || fired != "sync now" {
		t.Fatalf("sync: %d %q", code, fired)
	}
	if code, _ := get("GET", "qr.png", s.key); code != http.StatusNotFound {
		t.Fatalf("qr without sign-in: %d", code)
	}
}
