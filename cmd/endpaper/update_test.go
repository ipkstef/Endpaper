package main

import "testing"

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
