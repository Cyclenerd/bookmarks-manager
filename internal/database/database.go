// Package database manages the SQLite connection and schema initialisation.
package database

import (
	"database/sql"
	"fmt"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// schema contains the full DDL for the application. It is idempotent and can
// be executed on every startup. Triggers reset created_at on update, matching
// the original application's behaviour (created_at doubles as updated_at).
const schema = `
CREATE TABLE IF NOT EXISTS folders (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    parent_id TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (parent_id) REFERENCES folders (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS bookmarks (
    id TEXT PRIMARY KEY,
    url TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    favicon TEXT,
    folder_id TEXT,
    pinned INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (folder_id) REFERENCES folders (id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS bookmark_tags (
    bookmark_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    PRIMARY KEY (bookmark_id, tag_id),
    FOREIGN KEY (bookmark_id) REFERENCES bookmarks (id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_folder ON bookmarks(folder_id);
CREATE INDEX IF NOT EXISTS idx_bookmarks_pinned ON bookmarks(pinned);
CREATE INDEX IF NOT EXISTS idx_bookmark_tags_bookmark ON bookmark_tags(bookmark_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_tags_tag ON bookmark_tags(tag_id);

CREATE TRIGGER IF NOT EXISTS update_folders_timestamp
AFTER UPDATE ON folders
FOR EACH ROW
BEGIN
    UPDATE folders SET created_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_tags_timestamp
AFTER UPDATE ON tags
FOR EACH ROW
BEGIN
    UPDATE tags SET created_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_bookmarks_timestamp
AFTER UPDATE ON bookmarks
FOR EACH ROW
BEGIN
    UPDATE bookmarks SET created_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
`

// Open opens (creating if necessary) the SQLite database at path, enables
// foreign-key enforcement, and applies the schema.
//
// A path of ":memory:" is honoured for testing. The returned *sql.DB is safe
// for concurrent use; a single connection is used to keep an in-memory
// database coherent across goroutines.
func Open(path string) (*sql.DB, error) {
	dsn := path
	if path != ":memory:" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve db path: %w", err)
		}
		dsn = abs
	}
	// _pragma foreign_keys(1) ensures cascades/set-null work at the DB level.
	db, err := sql.Open("sqlite", dsn+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if path == ":memory:" {
		// Keep a single connection so the in-memory DB persists.
		db.SetMaxOpenConns(1)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := Init(db); err != nil {
		return nil, err
	}
	return db, nil
}

// Init applies the schema to an already-open database. It is idempotent.
func Init(db *sql.DB) error {
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}
