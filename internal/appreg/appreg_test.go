package appreg

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// newRegistry makes a file with the three tables we touch (no unique
// index on properties, the worst case).
func newRegistry(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "appreg.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, q := range []string{
		`CREATE TABLE interfaces(interface TEXT PRIMARY KEY)`,
		`CREATE TABLE handlerIds(handlerId TEXT PRIMARY KEY)`,
		`CREATE TABLE properties(handlerId TEXT, name TEXT, value TEXT)`,
		`INSERT INTO handlerIds VALUES('com.lab126.store')`,
		`INSERT INTO properties VALUES('com.lab126.store', 'command', 'x')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func count(t *testing.T, p, q string, args ...any) int {
	t.Helper()
	db, _ := sql.Open("sqlite", p)
	defer db.Close()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestRegister(t *testing.T) {
	p := newRegistry(t)
	bak := filepath.Join(t.TempDir(), "appreg.bak")
	props := map[string]string{"command": "/usr/bin/mesquite -l a.b -c file:///x/", "lipcId": "a.b"}
	changed, err := Register(p, "a.b", props, bak)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	if _, err := os.Stat(bak); err != nil {
		t.Fatal("no backup")
	}
	// Same again: no write, no duplicate rows.
	if changed, err = Register(p, "a.b", props, bak); err != nil || changed {
		t.Fatal(changed, err)
	}
	props["command"] = "new"
	if changed, _ = Register(p, "a.b", props, bak); !changed {
		t.Fatal("change not written")
	}
	if n := count(t, p, `SELECT count(*) FROM properties WHERE handlerId='a.b'`); n != 2 {
		t.Fatalf("%d property rows", n)
	}
	if err := Unregister(p, "a.b"); err != nil {
		t.Fatal(err)
	}
	if n := count(t, p, `SELECT count(*) FROM properties WHERE handlerId='a.b'`) +
		count(t, p, `SELECT count(*) FROM handlerIds WHERE handlerId='a.b'`); n != 0 {
		t.Fatalf("%d rows left", n)
	}
	// Other apps untouched.
	if n := count(t, p, `SELECT count(*) FROM properties WHERE handlerId='com.lab126.store'`); n != 1 {
		t.Fatal("store row lost")
	}
}

func TestNoCreate(t *testing.T) {
	p := filepath.Join(t.TempDir(), "missing.db")
	if _, err := Register(p, "a.b", map[string]string{"x": "y"}, ""); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("created the Kindle's file")
	}
}
