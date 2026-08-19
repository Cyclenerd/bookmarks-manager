package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
)

// BookmarkRepository provides CRUD, filtering and pagination for bookmarks.
type BookmarkRepository struct {
	db      *sql.DB
	folders *FolderRepository
}

// NewBookmarkRepository returns a BookmarkRepository backed by db. The folder
// repository is used to expand folder subtrees when requested.
func NewBookmarkRepository(db *sql.DB, folders *FolderRepository) *BookmarkRepository {
	return &BookmarkRepository{db: db, folders: folders}
}

// ListOptions controls filtering, sorting and pagination for GetAll.
type ListOptions struct {
	FolderID          string // folder UUID, or "unfiled", or "" for all
	TagID             string
	Search            string
	SortBy            string // title | url | created_at
	SortOrder         string // asc | desc
	IncludeSubfolders bool
	Page              int
	PerPage           int
}

var sortColumns = map[string]string{
	"title":      "b.title",
	"url":        "b.url",
	"created_at": "b.created_at",
}

// GetAll returns a paginated, filtered and sorted set of bookmarks. Pinned
// bookmarks always sort first, matching the original behaviour.
func (r *BookmarkRepository) GetAll(opts ListOptions) (*models.BookmarkPage, error) {
	if opts.Page < 1 {
		opts.Page = 1
	}
	if opts.PerPage < 1 {
		opts.PerPage = 25
	}

	isUnfiled := opts.FolderID == "unfiled"

	var folderIDs []string
	if opts.FolderID != "" && !isUnfiled {
		if opts.IncludeSubfolders {
			ids, err := r.folders.DescendantIDs(opts.FolderID)
			if err != nil {
				return nil, err
			}
			folderIDs = ids
		} else {
			folderIDs = []string{opts.FolderID}
		}
	}

	// Build shared WHERE clause and args.
	var where strings.Builder
	where.WriteString(" WHERE 1=1")
	var args []any

	if isUnfiled {
		where.WriteString(" AND b.folder_id IS NULL")
	} else if len(folderIDs) > 0 {
		where.WriteString(" AND b.folder_id IN (" + placeholders(len(folderIDs)) + ")")
		for _, id := range folderIDs {
			args = append(args, id)
		}
	}

	if opts.TagID != "" {
		where.WriteString(" AND b.id IN (SELECT bookmark_id FROM bookmark_tags WHERE tag_id = ?)")
		args = append(args, opts.TagID)
	}

	if opts.Search != "" {
		where.WriteString(" AND (b.title LIKE ? OR b.url LIKE ? OR b.description LIKE ?)")
		term := "%" + opts.Search + "%"
		args = append(args, term, term, term)
	}

	// Count total matches.
	countQuery := `
        SELECT COUNT(DISTINCT b.id)
        FROM bookmarks b
        LEFT JOIN folders f ON b.folder_id = f.id
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id` + where.String()

	var total int
	if err := r.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	sortColumn := sortColumns[opts.SortBy]
	if sortColumn == "" {
		sortColumn = "b.created_at"
	}
	sortDir := "DESC"
	if opts.SortOrder == "asc" {
		sortDir = "ASC"
	}
	offset := (opts.Page - 1) * opts.PerPage

	query := `
        SELECT b.id, b.url, b.title, b.description, b.favicon, b.folder_id, b.pinned, b.created_at,
               f.name AS folder_name,
               GROUP_CONCAT(t.name, ',') AS tag_names,
               GROUP_CONCAT(t.id, ',') AS tag_ids
        FROM bookmarks b
        LEFT JOIN folders f ON b.folder_id = f.id
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id` + where.String() +
		" GROUP BY b.id" +
		fmt.Sprintf(" ORDER BY b.pinned DESC, %s %s", sortColumn, sortDir) +
		" LIMIT ? OFFSET ?"

	rowArgs := append(append([]any{}, args...), opts.PerPage, offset)
	rows, err := r.db.Query(query, rowArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bookmarks []models.Bookmark
	for rows.Next() {
		b, err := scanBookmarkRow(rows)
		if err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := (total + opts.PerPage - 1) / opts.PerPage
	return &models.BookmarkPage{
		Bookmarks:  bookmarks,
		Total:      total,
		Page:       opts.Page,
		PerPage:    opts.PerPage,
		TotalPages: totalPages,
	}, nil
}

// scanBookmarkRow scans a joined bookmark row (with aggregated tag columns).
func scanBookmarkRow(rows *sql.Rows) (models.Bookmark, error) {
	var (
		b          models.Bookmark
		desc       sql.NullString
		favicon    sql.NullString
		folderID   sql.NullString
		folderName sql.NullString
		tagNames   sql.NullString
		tagIDs     sql.NullString
		pinned     int
	)
	if err := rows.Scan(&b.ID, &b.URL, &b.Title, &desc, &favicon, &folderID,
		&pinned, &b.CreatedAt, &folderName, &tagNames, &tagIDs); err != nil {
		return b, err
	}
	b.Description = desc.String
	b.Favicon = favicon.String
	if folderID.Valid {
		id := folderID.String
		b.FolderID = &id
	}
	b.FolderName = folderName.String
	b.Pinned = pinned != 0

	if tagNames.Valid && tagNames.String != "" {
		names := strings.Split(tagNames.String, ",")
		ids := strings.Split(tagIDs.String, ",")
		for i := range names {
			if i < len(ids) {
				b.Tags = append(b.Tags, models.Tag{ID: ids[i], Name: names[i]})
			}
		}
	}
	return b, nil
}

// Get returns a single bookmark with its tags, or nil if not found.
func (r *BookmarkRepository) Get(id string) (*models.Bookmark, error) {
	var (
		b        models.Bookmark
		desc     sql.NullString
		favicon  sql.NullString
		folderID sql.NullString
		pinned   int
	)
	err := r.db.QueryRow(`
        SELECT id, url, title, description, favicon, folder_id, pinned, created_at
        FROM bookmarks WHERE id = ?`, id).
		Scan(&b.ID, &b.URL, &b.Title, &desc, &favicon, &folderID, &pinned, &b.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	b.Description = desc.String
	b.Favicon = favicon.String
	if folderID.Valid {
		fid := folderID.String
		b.FolderID = &fid
	}
	b.Pinned = pinned != 0

	rows, err := r.db.Query(`
        SELECT t.id, t.name FROM tags t
        JOIN bookmark_tags bt ON t.id = bt.tag_id
        WHERE bt.bookmark_id = ?`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		b.Tags = append(b.Tags, t)
	}
	return &b, rows.Err()
}

// Create inserts a new bookmark and its tag associations.
func (r *BookmarkRepository) Create(url, title, description string, folderID *string, tagIDs []string, favicon string) (string, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	id := uuid.NewString()
	if _, err := tx.Exec(`
        INSERT INTO bookmarks (id, url, title, description, folder_id, favicon)
        VALUES (?, ?, ?, ?, ?, ?)`,
		id, url, title, description, folderID, nullIfEmpty(favicon)); err != nil {
		return "", err
	}
	if err := insertTags(tx, id, tagIDs); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

// Update replaces a bookmark's fields and tag associations. The favicon is
// only updated when a non-empty value is supplied.
func (r *BookmarkRepository) Update(id, url, title, description string, folderID *string, tagIDs []string, favicon string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if favicon != "" {
		_, err = tx.Exec(`
            UPDATE bookmarks SET url = ?, title = ?, description = ?, folder_id = ?, favicon = ?
            WHERE id = ?`, url, title, description, folderID, favicon, id)
	} else {
		_, err = tx.Exec(`
            UPDATE bookmarks SET url = ?, title = ?, description = ?, folder_id = ?
            WHERE id = ?`, url, title, description, folderID, id)
	}
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM bookmark_tags WHERE bookmark_id = ?`, id); err != nil {
		return err
	}
	if err := insertTags(tx, id, tagIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// Delete removes a bookmark and its tag associations.
func (r *BookmarkRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM bookmark_tags WHERE bookmark_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM bookmarks WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// TogglePin flips a bookmark's pinned flag and returns the new value.
func (r *BookmarkRepository) TogglePin(id string) (bool, error) {
	var pinned int
	err := r.db.QueryRow(`SELECT pinned FROM bookmarks WHERE id = ?`, id).Scan(&pinned)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	newVal := 0
	if pinned == 0 {
		newVal = 1
	}
	if _, err := r.db.Exec(`UPDATE bookmarks SET pinned = ? WHERE id = ?`, newVal, id); err != nil {
		return false, err
	}
	return newVal == 1, nil
}

func insertTags(tx *sql.Tx, bookmarkID string, tagIDs []string) error {
	for _, tagID := range tagIDs {
		if tagID == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)`,
			bookmarkID, tagID); err != nil {
			return err
		}
	}
	return nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
