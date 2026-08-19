package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Cyclenerd/bookmarks-manager/internal/models"
	"github.com/Cyclenerd/bookmarks-manager/internal/repository"
)

// Firefox container/place type identifiers.
const (
	typeContainer = "text/x-moz-place-container"
	typePlace     = "text/x-moz-place"
)

// specialGUIDs are Firefox's built-in roots that should not become folders,
// though their children are still processed.
var specialGUIDs = map[string]bool{
	"root________": true,
	"menu________": true,
	"unfiled_____": true,
	"mobile______": true,
	"toolbar_____": true,
}

// firefoxNode mirrors a node in a Firefox bookmark backup JSON file.
type firefoxNode struct {
	GUID     string        `json:"guid"`
	Title    string        `json:"title"`
	Type     string        `json:"type"`
	URI      string        `json:"uri,omitempty"`
	Tags     string        `json:"tags,omitempty"`
	Annos    []firefoxAnno `json:"annos,omitempty"`
	Children []firefoxNode `json:"children,omitempty"`
}

type firefoxAnno struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FirefoxService handles importing from and exporting to Firefox JSON.
type FirefoxService struct {
	db        *sql.DB
	bookmarks *repository.BookmarkRepository
	folders   *repository.FolderRepository
	tags      *repository.TagRepository
	favicons  *FaviconService
}

// NewFirefoxService constructs a FirefoxService.
func NewFirefoxService(
	db *sql.DB,
	bookmarks *repository.BookmarkRepository,
	folders *repository.FolderRepository,
	tags *repository.TagRepository,
	favicons *FaviconService,
) *FirefoxService {
	return &FirefoxService{db: db, bookmarks: bookmarks, folders: folders, tags: tags, favicons: favicons}
}

// Export builds a Firefox-compatible bookmark tree and returns it encoded as
// indented JSON.
func (s *FirefoxService) Export() ([]byte, error) {
	folderRows, err := s.db.Query(`SELECT id, name, parent_id FROM folders ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer folderRows.Close()

	// Use pointer nodes throughout so that appending to Children mutates the
	// shared tree in place regardless of processing order.
	folderMap := map[string]*firefoxNode{}
	var order []string
	parentOf := map[string]*string{}

	for folderRows.Next() {
		var id, name string
		var parentID sql.NullString
		if err := folderRows.Scan(&id, &name, &parentID); err != nil {
			return nil, err
		}
		folderMap[id] = &firefoxNode{
			GUID:  id,
			Title: name,
			Type:  typeContainer,
		}
		order = append(order, id)
		if parentID.Valid {
			p := parentID.String
			parentOf[id] = &p
		} else {
			parentOf[id] = nil
		}
	}
	if err := folderRows.Err(); err != nil {
		return nil, err
	}

	bookmarkRows, err := s.db.Query(`
        SELECT b.id, b.url, b.title, b.description, b.folder_id,
               GROUP_CONCAT(t.name, ',') AS tag_names
        FROM bookmarks b
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id
        GROUP BY b.id
        ORDER BY b.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer bookmarkRows.Close()

	var unorganized []firefoxNode
	for bookmarkRows.Next() {
		var id, url, title string
		var desc, folderID, tagNames sql.NullString
		if err := bookmarkRows.Scan(&id, &url, &title, &desc, &folderID, &tagNames); err != nil {
			return nil, err
		}
		node := firefoxNode{GUID: id, Title: title, Type: typePlace, URI: url}
		if desc.Valid && desc.String != "" {
			node.Annos = []firefoxAnno{{Name: "bookmarkProperties/description", Value: desc.String}}
		}
		if tagNames.Valid && tagNames.String != "" {
			node.Tags = tagNames.String
		}
		if folderID.Valid {
			if parent, ok := folderMap[folderID.String]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		unorganized = append(unorganized, node)
	}
	if err := bookmarkRows.Err(); err != nil {
		return nil, err
	}

	// Nest child folders into their parents bottom-up so that a parent already
	// contains its grandchildren before it is itself embedded. Folders are
	// ordered by name; process deepest first by repeatedly resolving leaves.
	nested := map[string]bool{}
	for progress := true; progress; {
		progress = false
		for _, id := range order {
			if nested[id] {
				continue
			}
			// Only nest folders whose children folders are all resolved.
			if hasUnnestedChildFolder(id, order, parentOf, nested) {
				continue
			}
			parent := parentOf[id]
			if parent == nil {
				continue // root, handled later
			}
			if pnode, ok := folderMap[*parent]; ok {
				pnode.Children = append(pnode.Children, *folderMap[id])
				nested[id] = true
				progress = true
			}
		}
	}

	var rootChildren []firefoxNode
	for _, id := range order {
		if parentOf[id] == nil {
			rootChildren = append(rootChildren, *folderMap[id])
		}
	}

	toolbar := firefoxNode{
		GUID:     "toolbar_____",
		Title:    "Bookmarks Toolbar",
		Type:     typeContainer,
		Children: append(rootChildren, unorganized...),
	}
	root := firefoxNode{
		GUID:     "root________",
		Title:    "",
		Type:     typeContainer,
		Children: []firefoxNode{toolbar},
	}

	return json.MarshalIndent(root, "", "  ")
}

// hasUnnestedChildFolder reports whether id has any child folder that has not
// yet been nested into it.
func hasUnnestedChildFolder(id string, order []string, parentOf map[string]*string, nested map[string]bool) bool {
	for _, other := range order {
		if p := parentOf[other]; p != nil && *p == id && !nested[other] {
			return true
		}
	}
	return false
}

// Parse decodes Firefox JSON, returning a descriptive error on malformed input.
func (s *FirefoxService) Parse(content []byte) (*firefoxNode, error) {
	var node firefoxNode
	if err := json.Unmarshal(content, &node); err != nil {
		return nil, fmt.Errorf("Invalid JSON format: %w", err)
	}
	return &node, nil
}

// Import walks the Firefox tree creating folders, bookmarks and tags, and
// downloading favicons. It returns statistics about what was imported.
func (s *FirefoxService) Import(root *firefoxNode) (models.ImportStats, error) {
	stats := models.ImportStats{}
	if err := s.process(root, nil, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

func (s *FirefoxService) process(node *firefoxNode, parentID *string, stats *models.ImportStats) error {
	switch node.Type {
	case typeContainer:
		if specialGUIDs[node.GUID] {
			for i := range node.Children {
				if err := s.process(&node.Children[i], parentID, stats); err != nil {
					return err
				}
			}
			return nil
		}
		title := node.Title
		if title == "" {
			return nil
		}
		folderID, err := s.folders.Create(title, parentID)
		if err != nil {
			return err
		}
		stats.Folders++
		for i := range node.Children {
			if err := s.process(&node.Children[i], &folderID, stats); err != nil {
				return err
			}
		}

	case typePlace:
		if node.URI == "" {
			return nil
		}
		title := node.Title
		if title == "" {
			title = node.URI
		}
		var description string
		for _, a := range node.Annos {
			if a.Name == "bookmarkProperties/description" {
				description = a.Value
				break
			}
		}

		var tagIDs []string
		if node.Tags != "" {
			for _, name := range strings.Split(node.Tags, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				existing, err := s.tags.FindByName(name)
				if err != nil {
					return err
				}
				if existing != nil {
					tagIDs = append(tagIDs, existing.ID)
					continue
				}
				id, err := s.tags.Create(name)
				if err != nil {
					return err
				}
				tagIDs = append(tagIDs, id)
				stats.Tags++
			}
		}

		favicon := s.favicons.Download(node.URI)
		if _, err := s.bookmarks.Create(node.URI, title, description, parentID, tagIDs, favicon); err != nil {
			return err
		}
		stats.Bookmarks++
	}
	return nil
}
