// Package appreg registers our Mesquite page as a Kindle app in the
// Kindle's app registry (/var/local/appreg.db, SQLite), so appmgrd can
// start it with "app://<id>". This is the only write to a Kindle system
// file (user decision 2026-10-24). We only add and remove our own rows,
// and a copy of the file is kept before the first write.
//
// Format: KindleModding wiki "WAFs & Mesquite" and "appreg.db";
// github.com/penguins184/IllusionChess (same three tables).
package appreg

import (
	"database/sql"
	"fmt"
	"io"
	"os"

	_ "modernc.org/sqlite"
)

// DefaultPath is the Kindle's app registry.
const DefaultPath = "/var/local/appreg.db"

// Register adds or updates the app id with the given properties (e.g.
// "command", "lipcId"). Rows are only written when a value differs, so a
// tap on an unchanged install writes nothing. backup, if not empty and not
// there yet, gets a copy of the file before the first write.
func Register(path, id string, props map[string]string, backup string) (changed bool, err error) {
	db, err := open(path)
	if err != nil {
		return false, err
	}
	defer db.Close()
	have := map[string]string{}
	rows, err := db.Query(`SELECT name, value FROM properties WHERE handlerId = ?`, id)
	if err != nil {
		return false, fmt.Errorf("appreg: %w", err)
	}
	for rows.Next() {
		var n, v string
		if err := rows.Scan(&n, &v); err != nil {
			rows.Close()
			return false, fmt.Errorf("appreg: %w", err)
		}
		have[n] = v
	}
	rows.Close()
	var hasID int
	_ = db.QueryRow(`SELECT count(*) FROM handlerIds WHERE handlerId = ?`, id).Scan(&hasID)
	same := hasID > 0 && len(have) == len(props)
	for k, v := range props {
		if have[k] != v {
			same = false
		}
	}
	if same {
		return false, nil
	}
	if backup != "" {
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			if err := copyFile(path, backup); err != nil {
				return false, fmt.Errorf("appreg: backup: %w", err)
			}
		}
	}
	tx, err := db.Begin()
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	// Delete + insert, not INSERT OR REPLACE: that needs a unique index on
	// (handlerId, name), which we do not know exists on every firmware.
	stmts := []struct {
		q    string
		args []any
	}{
		{`INSERT OR IGNORE INTO interfaces(interface) VALUES('application')`, nil},
		{`INSERT OR IGNORE INTO handlerIds(handlerId) VALUES(?)`, []any{id}},
		{`DELETE FROM properties WHERE handlerId = ?`, []any{id}},
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s.q, s.args...); err != nil {
			return false, fmt.Errorf("appreg: %w", err)
		}
	}
	for k, v := range props {
		if _, err := tx.Exec(`INSERT INTO properties(handlerId, name, value) VALUES(?, ?, ?)`, id, k, v); err != nil {
			return false, fmt.Errorf("appreg: %w", err)
		}
	}
	return true, tx.Commit()
}

// Unregister removes all rows of the app id.
func Unregister(path, id string) error {
	db, err := open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, q := range []string{
		`DELETE FROM properties WHERE handlerId = ?`,
		`DELETE FROM handlerIds WHERE handlerId = ?`,
	} {
		if _, err := tx.Exec(q, id); err != nil {
			return fmt.Errorf("appreg: %w", err)
		}
	}
	return tx.Commit()
}

func open(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("appreg: %w", err) // never create the Kindle's file
	}
	db, err := sql.Open("sqlite", "file:"+path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}
