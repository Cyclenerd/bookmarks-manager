// Package repository provides data-access logic for the application's models.
package repository

import (
	"database/sql"

	"github.com/google/uuid"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
)

// TagRepository provides CRUD operations for tags.
type TagRepository struct {
	db *sql.DB
}

// NewTagRepository returns a TagRepository backed by db.
func NewTagRepository(db *sql.DB) *TagRepository {
	return &TagRepository{db: db}
}

// GetAll returns every tag with its bookmark count, ordered by name.
func (r *TagRepository) GetAll() ([]models.Tag, error) {
	rows, err := r.db.Query(`
        SELECT t.id, t.name, t.created_at, COUNT(bt.bookmark_id) AS bookmark_count
        FROM tags t
        LEFT JOIN bookmark_tags bt ON t.id = bt.tag_id
        GROUP BY t.id
        ORDER BY t.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.BookmarkCount); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

// Get returns a single tag by ID, or nil if it does not exist.
func (r *TagRepository) Get(id string) (*models.Tag, error) {
	var t models.Tag
	err := r.db.QueryRow(`SELECT id, name, created_at FROM tags WHERE id = ?`, id).
		Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// FindByName returns the tag with the given name, or nil if none exists.
func (r *TagRepository) FindByName(name string) (*models.Tag, error) {
	var t models.Tag
	err := r.db.QueryRow(`SELECT id, name, created_at FROM tags WHERE name = ?`, name).
		Scan(&t.ID, &t.Name, &t.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// Create inserts a new tag and returns its generated ID.
func (r *TagRepository) Create(name string) (string, error) {
	id := uuid.NewString()
	if _, err := r.db.Exec(`INSERT INTO tags (id, name) VALUES (?, ?)`, id, name); err != nil {
		return "", err
	}
	return id, nil
}

// Update renames an existing tag.
func (r *TagRepository) Update(id, name string) error {
	_, err := r.db.Exec(`UPDATE tags SET name = ? WHERE id = ?`, name, id)
	return err
}

// Delete removes a tag and all of its bookmark associations.
func (r *TagRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM bookmark_tags WHERE tag_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
