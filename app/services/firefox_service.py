"""Firefox bookmark import/export service module.

This module provides functionality for importing and exporting bookmarks
in Firefox JSON format, enabling easy migration between Firefox and this application.
"""

from app.utils.database import get_db
from app.services import bookmark_service, folder_service, tag_service
import json


def export_to_firefox_json():
    """Export all bookmarks to Firefox JSON format.

    Builds a hierarchical structure compatible with Firefox bookmark JSON format,
    including all folders, bookmarks, tags, and descriptions.

    Returns:
        dict: Firefox-compatible bookmark structure with:
            - Root container
            - Toolbar container with folders and bookmarks
            - Nested folder hierarchy
            - Bookmark annotations for descriptions
            - Tag associations

    Note:
        Bookmarks without folders are placed in unorganized_bookmarks.
        All data is ordered chronologically by creation date.
    """
    db = get_db()

    # Get all folders
    folders_query = 'SELECT * FROM folders ORDER BY name'
    folders = db.execute(folders_query).fetchall()

    # Get all bookmarks
    bookmarks_query = '''
        SELECT b.*, f.name as folder_name,
               GROUP_CONCAT(t.name, ',') as tag_names
        FROM bookmarks b
        LEFT JOIN folders f ON b.folder_id = f.id
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id
        GROUP BY b.id
        ORDER BY b.created_at DESC
    '''
    bookmarks = db.execute(bookmarks_query).fetchall()

    # Build folder hierarchy
    folder_map = {}
    root_children = []

    # Create folder nodes
    for folder in folders:
        folder_node = {
            'guid': folder['id'],
            'title': folder['name'],
            'type': 'text/x-moz-place-container',
            'children': []
        }
        folder_map[folder['id']] = folder_node

        if folder['parent_id']:
            if folder['parent_id'] in folder_map:
                folder_map[folder['parent_id']]['children'].append(folder_node)
        else:
            root_children.append(folder_node)

    # Add bookmarks to their folders
    unorganized_bookmarks = []
    for bookmark in bookmarks:
        bookmark_node = {
            'guid': bookmark['id'],
            'title': bookmark['title'],
            'type': 'text/x-moz-place',
            'uri': bookmark['url']
        }

        if bookmark['description']:
            bookmark_node['annos'] = [{
                'name': 'bookmarkProperties/description',
                'value': bookmark['description']
            }]

        if bookmark['tag_names']:
            bookmark_node['tags'] = bookmark['tag_names']

        if bookmark['folder_id'] and bookmark['folder_id'] in folder_map:
            folder_map[bookmark['folder_id']]['children'].append(bookmark_node)
        else:
            unorganized_bookmarks.append(bookmark_node)

    # Build root structure
    toolbar = {
        'guid': 'toolbar_____',
        'title': 'Bookmarks Toolbar',
        'type': 'text/x-moz-place-container',
        'children': root_children + unorganized_bookmarks
    }

    root = {
        'guid': 'root________',
        'title': '',
        'type': 'text/x-moz-place-container',
        'children': [toolbar]
    }

    return root


def import_from_firefox_json(json_data):
    """Import bookmarks from Firefox JSON format.

    Recursively processes Firefox bookmark structure and imports:
    - Folders (excluding special Firefox containers)
    - Bookmarks with URLs, titles, and descriptions
    - Tags (creates new tags as needed)
    - Favicons (downloads automatically)

    Args:
        json_data (dict): Parsed Firefox JSON bookmark structure

    Returns:
        dict: Import statistics containing:
            - bookmarks (int): Number of bookmarks imported
            - folders (int): Number of folders created
            - tags (int): Number of new tags created

    Note:
        Skips special Firefox folders: root, menu, unfiled, mobile, toolbar.
        Automatically downloads favicons for imported bookmarks.
        Creates new tags if they don't exist in the database.
    """
    db = get_db()
    folder_mapping = {}
    stats = {'bookmarks': 0, 'folders': 0, 'tags': 0}

    def process_container(node, parent_id=None):
        """Recursively process Firefox bookmark containers."""
        if node.get('type') == 'text/x-moz-place-container':
            # Skip special Firefox folders we don't want to import
            guid = node.get('guid', '')
            if guid in ['root________', 'menu________', 'unfiled_____', 'mobile______', 'toolbar_____']:
                # Process children but don't create folder
                for child in node.get('children', []):
                    process_container(child, parent_id)
                return

            # Create folder
            folder_title = node.get('title', 'Untitled Folder')
            if folder_title:  # Only create if has a title
                folder_id = folder_service.create_folder(folder_title, parent_id)
                folder_mapping[node.get('guid')] = folder_id
                stats['folders'] += 1

                # Process children
                for child in node.get('children', []):
                    process_container(child, folder_id)

        elif node.get('type') == 'text/x-moz-place':
            # Import bookmark
            url = node.get('uri')
            title = node.get('title', url)

            if not url:
                return

            # Get description from annotations
            description = ''
            annos = node.get('annos', [])
            for anno in annos:
                if anno.get('name') == 'bookmarkProperties/description':
                    description = anno.get('value', '')
                    break

            # Handle tags
            tag_ids = []
            tags_str = node.get('tags', '')
            if tags_str:
                tag_names = tags_str.split(',') if isinstance(tags_str, str) else tags_str
                for tag_name in tag_names:
                    tag_name = tag_name.strip()
                    if tag_name:
                        # Find or create tag
                        tag = db.execute('SELECT id FROM tags WHERE name = ?', (tag_name,)).fetchone()
                        if tag:
                            tag_ids.append(tag['id'])
                        else:
                            tag_id = tag_service.create_tag(tag_name)
                            tag_ids.append(tag_id)
                            stats['tags'] += 1

            # Create bookmark
            from app.services.favicon_service import download_favicon
            favicon = download_favicon(url)
            bookmark_service.create_bookmark(url, title, description, parent_id, tag_ids, favicon)
            stats['bookmarks'] += 1

    # Start processing from root
    if isinstance(json_data, dict):
        process_container(json_data)

    return stats


def parse_firefox_json(file_content):
    """Parse Firefox JSON bookmark file content.

    Args:
        file_content (str): Raw JSON file content as string

    Returns:
        dict: Parsed JSON data structure

    Raises:
        ValueError: If JSON is malformed or cannot be parsed
    """
    try:
        data = json.loads(file_content)
        return data
    except json.JSONDecodeError as e:
        raise ValueError(f"Invalid JSON format: {str(e)}")
