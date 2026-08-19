package repository

import (
	"database/sql"
	"errors"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
)

// ErrCircularFolder is returned when a folder would be moved into one of its
// own descendants.
var ErrCircularFolder = errors.New("Cannot move folder into its own subfolder!")

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]`)

// FolderRepository provides CRUD and hierarchy operations for folders.
type FolderRepository struct {
	db *sql.DB
}

// NewFolderRepository returns a FolderRepository backed by db.
func NewFolderRepository(db *sql.DB) *FolderRepository {
	return &FolderRepository{db: db}
}

// stripForSort removes non-alphanumeric characters (e.g. leading emoji) and
// lowercases the result, used purely as a sort key.
func stripForSort(s string) string {
	return strings.ToLower(nonAlphaNum.ReplaceAllString(s, ""))
}

// flatFolders loads every folder with its bookmark and subfolder counts.
func (r *FolderRepository) flatFolders() ([]*models.Folder, error) {
	rows, err := r.db.Query(`
        SELECT f.id, f.name, f.parent_id, f.created_at,
               COUNT(DISTINCT b.id) AS bookmark_count,
               (SELECT COUNT(*) FROM folders WHERE parent_id = f.id) AS subfolder_count
        FROM folders f
        LEFT JOIN bookmarks b ON f.id = b.folder_id
        GROUP BY f.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*models.Folder
	for rows.Next() {
		f := &models.Folder{}
		if err := rows.Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedAt,
			&f.BookmarkCount, &f.SubfolderCount); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

// GetHierarchy builds and returns the folder tree. Root folders are returned
// at the top level, with descendants nested under Children. Siblings are
// sorted by name ignoring non-alphanumeric characters.
func (r *FolderRepository) GetHierarchy() ([]*models.Folder, error) {
	folders, err := r.flatFolders()
	if err != nil {
		return nil, err
	}

	byID := make(map[string]*models.Folder, len(folders))
	for _, f := range folders {
		f.Children = []*models.Folder{}
		byID[f.ID] = f
	}

	var roots []*models.Folder
	for _, f := range folders {
		if f.ParentID == nil {
			roots = append(roots, f)
		} else if parent, ok := byID[*f.ParentID]; ok {
			parent.Children = append(parent.Children, f)
		}
	}

	var sortRec func(list []*models.Folder)
	sortRec = func(list []*models.Folder) {
		sort.SliceStable(list, func(i, j int) bool {
			return stripForSort(list[i].Name) < stripForSort(list[j].Name)
		})
		for _, f := range list {
			if len(f.Children) > 0 {
				sortRec(f.Children)
			}
		}
	}
	sortRec(roots)
	return roots, nil
}

// Get returns a single folder including its parent chain (root to immediate
// parent), or nil if the folder does not exist.
func (r *FolderRepository) Get(id string) (*models.Folder, error) {
	f := &models.Folder{}
	err := r.db.QueryRow(`SELECT id, name, parent_id, created_at FROM folders WHERE id = ?`, id).
		Scan(&f.ID, &f.Name, &f.ParentID, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	current := f.ParentID
	for current != nil {
		p := &models.Folder{}
		err := r.db.QueryRow(`SELECT id, name, parent_id, created_at FROM folders WHERE id = ?`, *current).
			Scan(&p.ID, &p.Name, &p.ParentID, &p.CreatedAt)
		if err == sql.ErrNoRows {
			break
		}
		if err != nil {
			return nil, err
		}
		// Prepend to keep root-to-parent ordering.
		f.ParentChain = append([]*models.Folder{p}, f.ParentChain...)
		current = p.ParentID
	}
	return f, nil
}

// DescendantIDs returns id plus the IDs of all its descendant folders.
func (r *FolderRepository) DescendantIDs(id string) ([]string, error) {
	ids := []string{id}
	children, err := r.childIDs(id)
	if err != nil {
		return nil, err
	}
	for _, c := range children {
		sub, err := r.DescendantIDs(c)
		if err != nil {
			return nil, err
		}
		ids = append(ids, sub...)
	}
	return ids, nil
}

func (r *FolderRepository) childIDs(parentID string) ([]string, error) {
	rows, err := r.db.Query(`SELECT id FROM folders WHERE parent_id = ?`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Create inserts a new folder and returns its generated ID.
func (r *FolderRepository) Create(name string, parentID *string) (string, error) {
	id := uuid.NewString()
	if _, err := r.db.Exec(`INSERT INTO folders (id, name, parent_id) VALUES (?, ?, ?)`,
		id, name, parentID); err != nil {
		return "", err
	}
	return id, nil
}

// Update renames and/or reparents a folder. It returns ErrCircularFolder if
// the new parent is the folder itself or one of its descendants.
func (r *FolderRepository) Update(id, name string, parentID *string) error {
	if parentID != nil {
		descendants, err := r.DescendantIDs(id)
		if err != nil {
			return err
		}
		for _, d := range descendants {
			if d == *parentID {
				return ErrCircularFolder
			}
		}
	}
	_, err := r.db.Exec(`UPDATE folders SET name = ?, parent_id = ? WHERE id = ?`, name, parentID, id)
	return err
}

// Delete removes a folder, detaching (not deleting) any bookmarks it contains
// and cascading to subfolders via the foreign-key relationship.
func (r *FolderRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE bookmarks SET folder_id = NULL WHERE folder_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM folders WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}
