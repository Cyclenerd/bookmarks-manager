"""Database utility module.

This module provides database connection management, initialization, and
teardown functions for the SQLite database used by the bookmarks application.
"""

import sqlite3
from flask import current_app, g


def get_db():
    """Get the database connection for the current application context.

    Creates a new database connection if one doesn't exist in the application
    context. The connection uses Row factory for dict-like access to results.

    Returns:
        sqlite3.Connection: Database connection with Row factory

    Note:
        Connection is stored in Flask's g object and reused within the same request.
    """
    if 'db' not in g:
        db_path = current_app.config['DATABASE_PATH']
        check_same_thread = db_path == ':memory:' or current_app.config.get('TESTING', False)
        g.db = sqlite3.connect(
            db_path,
            check_same_thread=check_same_thread
        )
        g.db.row_factory = sqlite3.Row
    return g.db


def close_db(e=None):
    """Close the database connection for the current application context.

    This function is called automatically at the end of each request via
    Flask's teardown_appcontext handler.

    Args:
        e: Optional exception that triggered the teardown (unused)
    """
    db = g.pop('db', None)
    if db is not None:
        db.close()


def init_db():
    """Initialize the database schema.

    Creates all necessary tables and indexes if they don't exist. This includes:
    - folders: Hierarchical folder structure
    - tags: Tag definitions
    - bookmarks: Bookmark entries with URLs, titles, and metadata
    - bookmark_tags: Many-to-many relationship between bookmarks and tags
    - triggers: Automatic update of created_at timestamps on record modification

    All tables use UUID strings as primary keys for portability.

    Note:
        This function is idempotent and safe to call multiple times.
    """
    db = get_db()
    db.executescript('''
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
    ''')
    db.commit()


def teardown_db():
    """Teardown database connection.

    Alias for close_db() for consistency with other frameworks.
    """
    close_db()
