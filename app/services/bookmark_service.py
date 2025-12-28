"""Bookmark service module.

This module provides business logic for bookmark operations including
retrieval, creation, updating, deletion, and pinning functionality.
"""

from app.utils.database import get_db
import uuid


def get_all_bookmarks(folder_id=None, tag_id=None, search=None, sort_by='created_at', sort_order='desc',
                      include_subfolders=True, page=1, per_page=25):
    """Retrieve all bookmarks with optional filtering, sorting, and pagination.

    Args:
        folder_id (str, optional): Filter by folder UUID. If include_subfolders is True,
            includes bookmarks from all descendant folders. Use 'unfiled' to get bookmarks without a folder.
        tag_id (str, optional): Filter by tag UUID
        search (str, optional): Full-text search across title, URL, and description
        sort_by (str): Sort field - 'title', 'url', or 'created_at' (default: 'created_at')
        sort_order (str): Sort order - 'asc' or 'desc' (default: 'desc')
        include_subfolders (bool): Include descendant folders when folder_id is set (default: True)
        page (int): Page number for pagination (default: 1)
        per_page (int): Results per page (default: 25)

    Returns:
        dict: Dictionary containing:
            - bookmarks (list): List of bookmark dictionaries with tags
            - total (int): Total number of matching bookmarks
            - page (int): Current page number
            - per_page (int): Results per page
            - total_pages (int): Total number of pages

    Note:
        Pinned bookmarks always appear first regardless of sort order.
    """
    db = get_db()

    # Handle unfiled bookmarks (folder_id IS NULL)
    is_unfiled = folder_id == 'unfiled'

    folder_ids = []
    if folder_id and not is_unfiled:
        if include_subfolders:
            from app.services.folder_service import get_folder_with_descendants
            folder_ids = get_folder_with_descendants(folder_id)
        else:
            folder_ids = [folder_id]

    # Count total matching bookmarks
    count_query = '''
        SELECT COUNT(DISTINCT b.id)
        FROM bookmarks b
        LEFT JOIN folders f ON b.folder_id = f.id
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id
        WHERE 1=1
    '''
    count_params = []

    if is_unfiled:
        count_query += ' AND b.folder_id IS NULL'
    elif folder_ids:
        placeholders = ','.join('?' * len(folder_ids))
        count_query += f' AND b.folder_id IN ({placeholders})'
        count_params.extend(folder_ids)

    if tag_id:
        count_query += ' AND b.id IN (SELECT bookmark_id FROM bookmark_tags WHERE tag_id = ?)'
        count_params.append(tag_id)

    if search:
        count_query += ' AND (b.title LIKE ? OR b.url LIKE ? OR b.description LIKE ?)'
        search_term = f'%{search}%'
        count_params.extend([search_term, search_term, search_term])

    total = db.execute(count_query, count_params).fetchone()[0]

    query = '''
        SELECT DISTINCT b.*,
               f.name as folder_name,
               GROUP_CONCAT(t.name, ',') as tag_names,
               GROUP_CONCAT(t.id, ',') as tag_ids
        FROM bookmarks b
        LEFT JOIN folders f ON b.folder_id = f.id
        LEFT JOIN bookmark_tags bt ON b.id = bt.bookmark_id
        LEFT JOIN tags t ON bt.tag_id = t.id
        WHERE 1=1
    '''
    params = []

    if is_unfiled:
        query += ' AND b.folder_id IS NULL'
    elif folder_ids:
        placeholders = ','.join('?' * len(folder_ids))
        query += f' AND b.folder_id IN ({placeholders})'
        params.extend(folder_ids)

    if tag_id:
        query += ' AND b.id IN (SELECT bookmark_id FROM bookmark_tags WHERE tag_id = ?)'
        params.append(tag_id)

    if search:
        query += ' AND (b.title LIKE ? OR b.url LIKE ? OR b.description LIKE ?)'
        search_term = f'%{search}%'
        params.extend([search_term, search_term, search_term])

    query += ' GROUP BY b.id'

    valid_sorts = {'title': 'b.title', 'url': 'b.url', 'created_at': 'b.created_at'}
    sort_column = valid_sorts.get(sort_by, 'b.created_at')
    sort_direction = 'ASC' if sort_order == 'asc' else 'DESC'

    query += f' ORDER BY b.pinned DESC, {sort_column} {sort_direction}'

    # Add pagination
    offset = (page - 1) * per_page
    query += f' LIMIT {per_page} OFFSET {offset}'

    cursor = db.execute(query, params)
    bookmarks = []
    for row in cursor.fetchall():
        bookmark = dict(row)
        bookmark['tags'] = []
        if bookmark['tag_names']:
            tag_names = bookmark['tag_names'].split(',')
            tag_ids = bookmark['tag_ids'].split(',')
            bookmark['tags'] = [{'id': tid, 'name': tname} for tid, tname in zip(tag_ids, tag_names)]
        bookmarks.append(bookmark)

    return {
        'bookmarks': bookmarks,
        'total': total,
        'page': page,
        'per_page': per_page,
        'total_pages': (total + per_page - 1) // per_page
    }


def get_bookmark(bookmark_id):
    """Retrieve a single bookmark by ID.

    Args:
        bookmark_id (str): UUID of the bookmark

    Returns:
        dict or None: Bookmark dictionary with tags list, or None if not found
    """
    db = get_db()
    bookmark = db.execute('SELECT * FROM bookmarks WHERE id = ?', (bookmark_id,)).fetchone()
    if not bookmark:
        return None

    bookmark = dict(bookmark)
    tags = db.execute('''
        SELECT t.id, t.name FROM tags t
        JOIN bookmark_tags bt ON t.id = bt.tag_id
        WHERE bt.bookmark_id = ?
    ''', (bookmark_id,)).fetchall()
    bookmark['tags'] = [dict(tag) for tag in tags]
    return bookmark


def create_bookmark(url, title, description, folder_id, tag_ids, favicon=None):
    """Create a new bookmark.

    Args:
        url (str): Bookmark URL
        title (str): Bookmark title
        description (str): Bookmark description
        folder_id (str or None): UUID of parent folder, or None for no folder
        tag_ids (list): List of tag UUIDs to associate with bookmark
        favicon (str, optional): Path to cached favicon image

    Returns:
        str: UUID of the newly created bookmark
    """
    db = get_db()
    bookmark_id = str(uuid.uuid4())
    db.execute(
        'INSERT INTO bookmarks (id, url, title, description, folder_id, favicon) VALUES (?, ?, ?, ?, ?, ?)',
        (bookmark_id, url, title, description, folder_id if folder_id else None, favicon)
    )

    if tag_ids:
        for tag_id in tag_ids:
            db.execute('INSERT INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)', (bookmark_id, tag_id))

    db.commit()
    return bookmark_id


def update_bookmark(bookmark_id, url, title, description, folder_id, tag_ids, favicon=None):
    """Update an existing bookmark.

    Replaces all bookmark data including tag associations.

    Args:
        bookmark_id (str): UUID of bookmark to update
        url (str): New bookmark URL
        title (str): New bookmark title
        description (str): New bookmark description
        folder_id (str or None): New folder UUID, or None for no folder
        tag_ids (list): New list of tag UUIDs (replaces existing tags)
        favicon (str, optional): New favicon path (only updates if provided)
    """
    db = get_db()

    if favicon:
        db.execute(
            'UPDATE bookmarks SET url = ?, title = ?, description = ?, folder_id = ?, favicon = ? WHERE id = ?',
            (url, title, description, folder_id if folder_id else None, favicon, bookmark_id)
        )
    else:
        db.execute(
            'UPDATE bookmarks SET url = ?, title = ?, description = ?, folder_id = ? WHERE id = ?',
            (url, title, description, folder_id if folder_id else None, bookmark_id)
        )

    db.execute('DELETE FROM bookmark_tags WHERE bookmark_id = ?', (bookmark_id,))
    if tag_ids:
        for tag_id in tag_ids:
            db.execute('INSERT INTO bookmark_tags (bookmark_id, tag_id) VALUES (?, ?)', (bookmark_id, tag_id))

    db.commit()


def delete_bookmark(bookmark_id):
    """Delete a bookmark.

    Also removes all associated tag relationships.

    Args:
        bookmark_id (str): UUID of bookmark to delete
    """
    db = get_db()
    db.execute('DELETE FROM bookmarks WHERE id = ?', (bookmark_id,))
    db.commit()


def toggle_pin(bookmark_id):
    """Toggle the pinned status of a bookmark.

    Args:
        bookmark_id (str): UUID of bookmark to toggle

    Returns:
        int or None: New pinned value (0 or 1), or None if bookmark not found
    """
    db = get_db()
    bookmark = db.execute('SELECT pinned FROM bookmarks WHERE id = ?', (bookmark_id,)).fetchone()
    if bookmark:
        new_pinned = 0 if bookmark['pinned'] else 1
        db.execute('UPDATE bookmarks SET pinned = ? WHERE id = ?', (new_pinned, bookmark_id))
        db.commit()
        return new_pinned
    return None
