# Database

The application uses SQLite with the following tables:

- **folders**: Hierarchical folder structure (self-referencing with `parent_id`)
- **tags**: Simple tag list
- **bookmarks**: Main bookmark storage with `folder_id`, `pinned`, `favicon`, etc.
- **bookmark_tags**: Many-to-many relationship between bookmarks and tags

All tables use UUID primary keys for better security.
