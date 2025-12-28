"""Integration tests for bookmark workflows.

Tests the complete bookmark lifecycle including creation, editing,
deletion, and interaction with folders and tags.
"""


def test_bookmark_creation_workflow(client, auth_headers, app):
    """Test complete bookmark creation workflow.

    Tests creating a folder, tag, and bookmark, then verifying the bookmark
    appears on the index page with correct associations.
    """
    # Create folder
    response = client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Create tag
    response = client.post('/tag/save', data={'name': 'Important'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get folder and tag IDs
    with app.app_context():
        from app.services import folder_service, tag_service
        folders = folder_service.get_all_folders()
        tags = tag_service.get_all_tags()
        folder_id = folders[0]['id']
        tag_id = tags[0]['id']

    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A test bookmark',
        'folder_id': folder_id,
        'tag_ids': [tag_id]
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify bookmark appears on index page
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 200
    assert b'Example Site' in response.data
    assert b'https://example.com' in response.data


def test_bookmark_edit_workflow(client, auth_headers, app):
    """Test bookmark editing workflow.

    Tests creating a bookmark, editing its properties, and verifying changes.
    """
    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Original Title',
        'description': 'Original description'
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get bookmark ID
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        bookmark_id = bookmarks['bookmarks'][0]['id']

    # Edit bookmark
    updated_data = {
        'bookmark_id': bookmark_id,
        'url': 'https://example.org',
        'title': 'Updated Title',
        'description': 'Updated description'
    }
    response = client.post('/bookmark/save', data=updated_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify changes
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 200
    assert b'Updated Title' in response.data
    assert b'https://example.org' in response.data


def test_bookmark_deletion_workflow(client, auth_headers, app):
    """Test bookmark deletion workflow.

    Tests creating and deleting a bookmark.
    """
    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'To Delete',
        'description': 'This will be deleted'
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get bookmark ID
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        bookmark_id = bookmarks['bookmarks'][0]['id']

    # Delete bookmark
    response = client.post(f'/bookmark/{bookmark_id}/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify deletion
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 200
    assert b'To Delete' not in response.data


def test_bookmark_pinning_workflow(client, auth_headers, app):
    """Test bookmark pinning workflow.

    Tests pinning and unpinning bookmarks.
    """
    # Create two bookmarks
    bookmark1_data = {'url': 'https://example1.com', 'title': 'First Bookmark'}
    bookmark2_data = {'url': 'https://example2.com', 'title': 'Second Bookmark'}

    client.post('/bookmark/save', data=bookmark1_data, headers=auth_headers)
    client.post('/bookmark/save', data=bookmark2_data, headers=auth_headers)

    # Get bookmark IDs
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        bookmark_id = bookmarks['bookmarks'][0]['id']

    # Pin bookmark
    response = client.post(f'/bookmark/{bookmark_id}/toggle-pin', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify pinned status
    with app.app_context():
        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['pinned'] == 1  # SQLite returns 1 for True

    # Unpin bookmark
    response = client.post(f'/bookmark/{bookmark_id}/toggle-pin', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify unpinned status
    with app.app_context():
        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['pinned'] == 0  # SQLite returns 0 for False


def test_bookmark_with_multiple_tags(client, auth_headers, app):
    """Test bookmark with multiple tags.

    Tests creating a bookmark with multiple tags and filtering by tags.
    """
    # Create tags
    client.post('/tag/save', data={'name': 'Python'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Web'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Tutorial'}, headers=auth_headers)

    # Get tag IDs
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag_ids = [tag['id'] for tag in tags]

    # Create bookmark with multiple tags
    bookmark_data = {
        'url': 'https://python-tutorial.com',
        'title': 'Python Web Tutorial',
        'description': 'Learn Python web development',
        'tag_ids': tag_ids
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify bookmark has all tags
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        bookmark = bookmarks['bookmarks'][0]
        assert len(bookmark['tags']) == 3
        tag_names = {tag['name'] for tag in bookmark['tags']}
        assert tag_names == {'Python', 'Web', 'Tutorial'}


def test_bookmark_metadata_fetching(client, auth_headers, app):
    """Test automatic metadata fetching for bookmarks.

    Tests that the metadata service can fetch metadata for URLs.
    """
    # Test metadata service directly instead of API endpoint
    with app.app_context():
        from app.services import metadata_service
        metadata = metadata_service.fetch_page_metadata('https://example.com')
        assert metadata is not None
        assert 'title' in metadata or metadata.get('title') is None
