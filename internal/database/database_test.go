package database

import (
	"path/filepath"
	"testing"
)

func TestDefaultDurabilityPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "durable.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journal != "truncate" {
		t.Errorf("expected journal_mode=truncate (crash-safe on gcsfuse), got %q", journal)
	}

	var sync int
	if err := db.QueryRow("PRAGMA synchronous").Scan(&sync); err != nil {
		t.Fatalf("synchronous: %v", err)
	}
	if sync != 2 { // 2 == FULL
		t.Errorf("expected synchronous=FULL (2), got %d", sync)
	}

	var fk int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&fk); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if fk != 1 {
		t.Errorf("expected foreign_keys enabled, got %d", fk)
	}
}

func TestSingleConnectionOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "single.db")
	db, err := OpenWithOptions(path, Options{SingleConnection: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("expected MaxOpenConnections=1 for gcsfuse safety, got %d", got)
	}
}

func TestOverrideJournalMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.db")
	db, err := OpenWithOptions(path, Options{JournalMode: "WAL", Synchronous: "NORMAL"})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	var journal string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journal); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journal != "wal" {
		t.Errorf("expected overridden journal_mode=wal, got %q", journal)
	}
}

// TestDataPersistsAfterClose simulates the shutdown path: data written and then
// the handle closed must be readable when the database is reopened. This is the
// behaviour that makes gcsfuse upload the final state to Cloud Storage.
func TestDataPersistsAfterClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")

	db, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO tags (id, name) VALUES ('t1', 'go')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	db2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()

	var name string
	if err := db2.QueryRow(`SELECT name FROM tags WHERE id = 't1'`).Scan(&name); err != nil {
		t.Fatalf("read after reopen: %v", err)
	}
	if name != "go" {
		t.Errorf("expected persisted tag 'go', got %q", name)
	}
}
