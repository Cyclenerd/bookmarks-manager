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

// Options controls how the SQLite database is opened.
type Options struct {
	// JournalMode is the SQLite journal mode (WAL, TRUNCATE, DELETE, ...).
	// Empty selects a safe default: TRUNCATE, which works on gcsfuse.
	JournalMode string
	// Synchronous is the SQLite synchronous setting (FULL, NORMAL, OFF).
	// Empty selects FULL for maximum durability.
	Synchronous string
	// Durable, when true, forces a single writer connection. This is required
	// on gcsfuse-backed volumes, which do not provide concurrency control for
	// multiple writers to the same file ("last write wins" — see the package
	// notes). It is the safe default for the Cloud Run deployment.
	SingleConnection bool
}

// Open opens (creating if necessary) the SQLite database at path with
// durability-first defaults suitable for a gcsfuse-backed volume on a
// serverless runtime that can be killed at any time.
//
// A path of ":memory:" is honoured for testing.
func Open(path string) (*sql.DB, error) {
	return OpenWithOptions(path, Options{})
}

// OpenWithOptions opens the database applying the given Options.
//
// # Durability on gcsfuse (Cloud Run) where the instance can be killed anytime
//
// Cloud Storage FUSE (gcsfuse) is not fully POSIX-compliant and only persists
// a file to Cloud Storage when it is fsync()'d or closed. It also provides no
// locking for concurrent writers to the same file. SQLite, meanwhile, keeps
// the database file open for the whole process lifetime. If the instance is
// killed (SIGKILL after the shutdown grace period, or preemption) these two
// facts combine to threaten durability and integrity. The chosen settings
// mitigate this:
//
//   - journal_mode=TRUNCATE keeps a real on-disk rollback journal (unlike
//     MEMORY, which would leave the database unrecoverable after a crash mid
//     write). WAL is avoided because gcsfuse cannot back its shared-memory
//     index.
//   - synchronous=FULL fsync()s the database (and journal) at each commit.
//     On gcsfuse the fsync is exactly what flushes the committed transaction
//     up to Cloud Storage, so every completed write is durable the moment the
//     commit returns — minimising the data-loss window if the instance dies.
//   - A single writer connection avoids gcsfuse's "last write wins" behaviour
//     that could otherwise corrupt the file when two connections flush it.
//
// These favour correctness over raw throughput, which is the right trade-off
// for this single-user application.
func OpenWithOptions(path string, opts Options) (*sql.DB, error) {
	journalMode := opts.JournalMode
	if journalMode == "" {
		journalMode = "TRUNCATE"
	}
	synchronous := opts.Synchronous
	if synchronous == "" {
		synchronous = "FULL"
	}

	dsn := path
	if path != ":memory:" {
		abs, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve db path: %w", err)
		}
		dsn = abs
	}
	// Pragmas applied on every connection:
	//   foreign_keys(1)    ensures cascades / set-null work at the DB level.
	//   busy_timeout(5000) avoids "database is locked" errors under contention.
	//   journal_mode(...)  see the doc comment above.
	//   synchronous(...)   see the doc comment above.
	db, err := sql.Open("sqlite",
		fmt.Sprintf("%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"+
			"&_pragma=journal_mode(%s)&_pragma=synchronous(%s)", dsn, journalMode, synchronous))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// On a gcsfuse mount a single connection is mandatory for integrity; the
	// in-memory database also needs exactly one connection to persist. In both
	// cases cap the pool at one. Otherwise allow a small bounded pool.
	if opts.SingleConnection || path == ":memory:" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	} else {
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(2)
	}
	db.SetConnMaxIdleTime(0) // never proactively close idle conns

	// Running the schema (Init) opens the first real connection and validates
	// connectivity, so a separate db.Ping() would only add an extra round-trip
	// to a potentially slow (network-mounted) database file during cold start.
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
