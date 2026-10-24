package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewer(t *testing.T) {
	for _, c := range []struct {
		latest, current string
		want            bool
	}{
		{"v0.2.0", "v0.1.0", true},
		{"v0.1.1", "v0.1.0", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.10.0", "v0.9.0", true}, // numbers, not text
		{"v0.1.0", "v0.1.0", false},
		{"v0.1.0", "v0.2.0", false},
		{"v0.1.0", "v0.1.0-3-gabc123", false}, // build after the tag
		{"v0.2.0", "v0.1.0-3-gabc123", true},
		{"v0.2.0", "192913a", false}, // development build
		{"v0.2.0", "dev", false},
		{"", "v0.1.0", false},
		{"nightly", "v0.1.0", false},
	} {
		if got := newer(c.latest, c.current); got != c.want {
			t.Errorf("newer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

// The first status call starts a check and reports it as running; the
// next call shows the newer release. A failed check is retried sooner.
func TestUpdatesCheck(t *testing.T) {
	tag := `{"tag_name":"v0.2.0"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(tag))
	}))
	defer srv.Close()
	old := latestURL
	latestURL = srv.URL
	defer func() { latestURL = old }()

	u := &updates{http: srv.Client()}
	if got, checking := u.available("v0.1.0"); got != "" || !checking {
		t.Fatalf("first call: %q checking %v", got, checking)
	}
	for i := 0; i < 50; i++ {
		if _, checking := u.available("v0.1.0"); !checking {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got, _ := u.available("v0.1.0"); got != "v0.2.0" {
		t.Fatalf("got %q", got)
	}
	if got, checking := u.available("dev"); got != "" || checking {
		t.Fatal("development builds never check")
	}
}
