"""Folder service module.

This module provides business logic for folder operations including
hierarchical folder management, retrieval, and manipulation.
"""

from app.utils.database import get_db
import uuid
import re


def _strip_emoji_for_sort(text):
    """Remove all non-alphanumeric characters for sorting purposes.

    Args:
        text (str): Text that may contain emoji, symbols, or punctuation

    Returns:
        str: Text with only letters and numbers remaining
    """
    return re.sub(r'[^a-zA-Z0-9]', '', text)


def get_all_folders():
    """Retrieve all folders with bookmark and subfolder counts.

    Returns:
        list: List of folder dictionaries with:
            - id, name, parent_id, created_at
            - bookmark_count: Number of bookmarks directly in folder
            - subfolder_count: Number of immediate child folders

    Note:
        Results are ordered by parent_id and name for consistent display.
    """
    db = get_db()
    folders = db.execute('''
        SELECT f.*, COUNT(DISTINCT b.id) as bookmark_count,
               (SELECT COUNT(*) FROM folders WHERE parent_id = f.id) as subfolder_count
        FROM folders f
        LEFT JOIN bookmarks b ON f.id = b.folder_id
        GROUP BY f.id
        ORDER BY f.parent_id, f.name
    ''').fetchall()
    return [dict(folder) for folder in folders]


def get_folder_hierarchy():
    """Build a hierarchical tree of all folders.

    Returns:
        list: List of root folder dictionaries, each containing:
            - Standard folder fields (id, name, parent_id, created_at)
            - bookmark_count: Number of bookmarks in folder
            - subfolder_count: Number of immediate child folders
            - children: Recursively nested list of child folders

    Note:
        Only root folders (parent_id is None) are at the top level.
        All descendant folders are nested in the 'children' property.
        Folders are sorted alphabetically by name, ignoring leading emoji.
    """
    db = get_db()
    folders = db.execute('''
        SELECT f.*, COUNT(DISTINCT b.id) as bookmark_count,
               (SELECT COUNT(*) FROM folders WHERE parent_id = f.id) as subfolder_count
        FROM folders f
        LEFT JOIN bookmarks b ON f.id = b.folder_id
        GROUP BY f.id
    ''').fetchall()

    folder_dict = {f['id']: dict(f) for f in folders}
    for folder in folder_dict.values():
        folder['children'] = []

    root_folders = []
    for folder in folder_dict.values():
        if folder['parent_id'] is None:
            root_folders.append(folder)
        else:
            if folder['parent_id'] in folder_dict:
                folder_dict[folder['parent_id']]['children'].append(folder)

    def sort_folders_recursive(folder_list):
        """Sort folders by name (ignoring emoji) and recursively sort children."""
        folder_list.sort(key=lambda f: _strip_emoji_for_sort(f['name']).lower())
        for folder in folder_list:
            if folder['children']:
                sort_folders_recursive(folder['children'])

    sort_folders_recursive(root_folders)
    return root_folders


def get_folder(folder_id):
    """Retrieve a single folder by ID with parent chain.

    Args:
        folder_id (str): UUID of the folder

    Returns:
        dict or None: Folder dictionary with 'parent_chain' list containing
            all ancestor folders from root to immediate parent, or None if not found
    """
    db = get_db()
    folder = db.execute('SELECT * FROM folders WHERE id = ?', (folder_id,)).fetchone()
    if not folder:
        return None
    folder_dict = dict(folder)

    parent_chain = []
    current_parent_id = folder_dict['parent_id']
    while current_parent_id:
        parent = db.execute('SELECT * FROM folders WHERE id = ?', (current_parent_id,)).fetchone()
        if parent:
            parent_chain.insert(0, dict(parent))
            current_parent_id = parent['parent_id']
        else:
            break
    folder_dict['parent_chain'] = parent_chain
    return folder_dict


def get_folder_with_descendants(folder_id):
    """Get a folder and all its descendant folder IDs.

    Recursively collects all subfolder IDs including the specified folder.

    Args:
        folder_id (str): UUID of the root folder

    Returns:
        list: List of folder UUIDs including folder_id and all descendants
    """
    db = get_db()

    def get_descendants(parent_id):
        children = db.execute('SELECT id FROM folders WHERE parent_id = ?', (parent_id,)).fetchall()
        ids = [parent_id]
        for child in children:
            ids.extend(get_descendants(child['id']))
        return ids

    return get_descendants(folder_id)


def create_folder(name, parent_id=None):
    """Create a new folder.

    Args:
        name (str): Folder name
        parent_id (str, optional): UUID of parent folder, or None for root folder

    Returns:
        str: UUID of the newly created folder
    """
    db = get_db()
    folder_id = str(uuid.uuid4())
    db.execute('INSERT INTO folders (id, name, parent_id) VALUES (?, ?, ?)', (folder_id, name, parent_id))
    db.commit()
    return folder_id


def update_folder(folder_id, name, parent_id=None):
    """Update an existing folder.

    Args:
        folder_id (str): UUID of folder to update
        name (str): New folder name
        parent_id (str, optional): New parent folder UUID, or None for root

    Raises:
        ValueError: If attempting to move folder into its own subfolder
    """
    db = get_db()

    if parent_id:
        descendants = get_folder_with_descendants(folder_id)
        if parent_id in descendants:
            raise ValueError("Cannot move folder into its own subfolder")

    db.execute('UPDATE folders SET name = ?, parent_id = ? WHERE id = ?', (name, parent_id, folder_id))
    db.commit()


def delete_folder(folder_id):
    """Delete a folder.

    Bookmarks in the folder are set to have no folder (folder_id becomes NULL).

    Args:
        folder_id (str): UUID of folder to delete
    """
    db = get_db()
    db.execute('DELETE FROM folders WHERE id = ?', (folder_id,))
    db.commit()
