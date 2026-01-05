"""Tag service module.

This module provides business logic for tag operations including
creation, retrieval, updating, and deletion of bookmark tags.
"""

from app.utils.database import get_db
import uuid


def get_all_tags():
    """Retrieve all tags with bookmark counts.

    Returns:
        list: List of tag dictionaries with:
            - id, name, created_at
            - bookmark_count: Number of bookmarks using this tag

    Note:
        Results are ordered by tag name alphabetically.
    """
    db = get_db()
    tags = db.execute('''
        SELECT t.*, COUNT(bt.bookmark_id) as bookmark_count
        FROM tags t
        LEFT JOIN bookmark_tags bt ON t.id = bt.tag_id
        GROUP BY t.id
        ORDER BY t.name
    ''').fetchall()
    return [dict(tag) for tag in tags]


def get_tag(tag_id):
    """Retrieve a single tag by ID.

    Args:
        tag_id (str): UUID of the tag

    Returns:
        dict or None: Tag dictionary, or None if not found
    """
    db = get_db()
    tag = db.execute('SELECT * FROM tags WHERE id = ?', (tag_id,)).fetchone()
    return dict(tag) if tag else None


def create_tag(name):
    """Create a new tag.

    Args:
        name (str): Tag name (must be unique)

    Returns:
        str: UUID of the newly created tag
    """
    db = get_db()
    tag_id = str(uuid.uuid4())
    db.execute('INSERT INTO tags (id, name) VALUES (?, ?)', (tag_id, name))
    db.commit()
    return tag_id


def update_tag(tag_id, name):
    """Update an existing tag.

    Args:
        tag_id (str): UUID of tag to update
        name (str): New tag name (must be unique)
    """
    db = get_db()
    db.execute('UPDATE tags SET name = ? WHERE id = ?', (name, tag_id))
    db.commit()


def delete_tag(tag_id):
    """Delete a tag.

    Also removes all bookmark-tag associations from the bookmark_tags table.

    Args:
        tag_id (str): UUID of tag to delete
    """
    db = get_db()
    db.execute('DELETE FROM bookmark_tags WHERE tag_id = ?', (tag_id,))
    db.execute('DELETE FROM tags WHERE id = ?', (tag_id,))
    db.commit()
