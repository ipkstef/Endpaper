package main

import "testing"

func TestSecretNamesNeverCopied(t *testing.T) {
	for _, p := range []string{"/var/local/java/keystore/ks.jks", "/var/local/token.json", "/mnt/us/system/.assets/cookies.db-x",
		"/var/local/amazon/certs/home.pem", "/var/local/DSN"} {
		if !secret.MatchString(p) {
			t.Errorf("%s would be copied", p)
		}
	}
	for _, p := range []string{"/var/local/merchant/home_widgets.json", "/mnt/us/system/thumbnails/thumbnail_x.jpg"} {
		if secret.MatchString(p) || !interesting.MatchString(p) {
			t.Errorf("%s: secret %v interesting %v", p, secret.MatchString(p), interesting.MatchString(p))
		}
	}
}
