"""Integration tests for tag workflows.

Tests the complete tag lifecycle including creation, editing,
deletion, and tag-based bookmark filtering.
"""


def test_tag_creation_workflow(client, auth_headers, app):
    """Test tag creation workflow.

    Tests creating tags and verifying they appear in the system.
    """
    # Create tag
    response = client.post('/tag/save', data={'name': 'Python'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify tag exists
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        assert len(tags) == 1
        assert tags[0]['name'] == 'Python'


def test_multiple_tags_creation(client, auth_headers, app):
    """Test creating multiple tags.

    Tests creating several tags and verifying count.
    """
    # Create multiple tags
    tag_names = ['Python', 'JavaScript', 'Web', 'Tutorial', 'Documentation']
    for tag_name in tag_names:
        response = client.post('/tag/save', data={'name': tag_name}, headers=auth_headers)
        assert response.status_code == 302

    # Verify all tags exist
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        assert len(tags) == len(tag_names)
        retrieved_names = {tag['name'] for tag in tags}
        assert retrieved_names == set(tag_names)


def test_tag_edit_workflow(client, auth_headers, app):
    """Test tag editing workflow.

    Tests creating a tag, editing its name, and verifying changes.
    """
    # Create tag
    response = client.post('/tag/save', data={'name': 'Old Tag'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get tag ID
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag_id = tags[0]['id']

    # Edit tag name
    response = client.post('/tag/save',
                           data={'tag_id': tag_id, 'name': 'New Tag'},
                           headers=auth_headers,
                           follow_redirects=False)
    assert response.status_code == 302

    # Verify change
    with app.app_context():
        tag = tag_service.get_tag(tag_id)
        assert tag['name'] == 'New Tag'


def test_tag_deletion_workflow(client, auth_headers, app):
    """Test tag deletion workflow.

    Tests creating and deleting a tag.
    """
    # Create tag
    response = client.post('/tag/save', data={'name': 'To Delete'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get tag ID
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag_id = tags[0]['id']

    # Delete tag
    response = client.post(f'/tag/{tag_id}/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify deletion
    with app.app_context():
        tags = tag_service.get_all_tags()
        assert len(tags) == 0


def test_tag_with_bookmarks_deletion(client, auth_headers, app):
    """Test deleting tag with associated bookmarks.

    Tests that bookmark-tag associations are removed when tag is deleted.
    """
    # Create tag
    response = client.post('/tag/save', data={'name': 'Important'}, headers=auth_headers)
    assert response.status_code == 302

    # Get tag ID
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag_id = tags[0]['id']

    # Create bookmark with tag
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example',
        'tag_ids': [tag_id]
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)
    assert response.status_code == 302

    # Delete tag
    response = client.post(f'/tag/{tag_id}/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify bookmark still exists but without tag
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        assert len(bookmarks['bookmarks']) == 1
        assert len(bookmarks['bookmarks'][0]['tags']) == 0


def test_tag_filtering_workflow(client, auth_headers, app):
    """Test filtering bookmarks by tag.

    Tests creating tags and bookmarks, then filtering by tag.
    """
    # Create two tags
    client.post('/tag/save', data={'name': 'Python'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'JavaScript'}, headers=auth_headers)

    # Get tag IDs
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        python_id = [t['id'] for t in tags if t['name'] == 'Python'][0]
        javascript_id = [t['id'] for t in tags if t['name'] == 'JavaScript'][0]

    # Create bookmarks with different tags
    client.post('/bookmark/save',
                data={'url': 'https://python.org', 'title': 'Python Site', 'tag_ids': [python_id]},
                headers=auth_headers)
    client.post('/bookmark/save',
                data={'url': 'https://javascript.com', 'title': 'JS Site', 'tag_ids': [javascript_id]},
                headers=auth_headers)

    # Filter by Python tag
    response = client.get(f'/?tag={python_id}', headers=auth_headers)
    assert response.status_code == 200
    assert b'Python Site' in response.data
    assert b'JS Site' not in response.data

    # Filter by JavaScript tag
    response = client.get(f'/?tag={javascript_id}', headers=auth_headers)
    assert response.status_code == 200
    assert b'JS Site' in response.data
    assert b'Python Site' not in response.data


def test_tag_usage_tracking(client, auth_headers, app):
    """Test tag usage tracking.

    Tests that tags are correctly associated with bookmarks.
    """
    # Create tag
    client.post('/tag/save', data={'name': 'Popular'}, headers=auth_headers)

    # Get tag ID
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag_id = tags[0]['id']

    # Create bookmarks with the tag
    for i in range(3):
        client.post('/bookmark/save',
                    data={'url': f'https://example{i}.com', 'title': f'Site {i}', 'tag_ids': [tag_id]},
                    headers=auth_headers)

    # Verify bookmarks have the tag
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        tagged_count = sum(1 for b in bookmarks['bookmarks'] if any(t['id'] == tag_id for t in b['tags']))
        assert tagged_count == 3


def test_bookmark_tag_association_update(client, auth_headers, app):
    """Test updating bookmark tag associations.

    Tests adding and removing tags from an existing bookmark.
    """
    # Create tags
    client.post('/tag/save', data={'name': 'Tag1'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Tag2'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Tag3'}, headers=auth_headers)

    # Get tag IDs
    with app.app_context():
        from app.services import tag_service
        tags = tag_service.get_all_tags()
        tag1_id = [t['id'] for t in tags if t['name'] == 'Tag1'][0]
        tag2_id = [t['id'] for t in tags if t['name'] == 'Tag2'][0]
        tag3_id = [t['id'] for t in tags if t['name'] == 'Tag3'][0]

    # Create bookmark with Tag1 and Tag2
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example',
        'tag_ids': [tag1_id, tag2_id]
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Get bookmark ID
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        bookmark_id = bookmarks['bookmarks'][0]['id']

    # Update to Tag2 and Tag3 (remove Tag1, add Tag3)
    updated_data = {
        'bookmark_id': bookmark_id,
        'url': 'https://example.com',
        'title': 'Example',
        'tag_ids': [tag2_id, tag3_id]
    }
    client.post('/bookmark/save', data=updated_data, headers=auth_headers)

    # Verify tag associations
    with app.app_context():
        bookmark = bookmark_service.get_bookmark(bookmark_id)
        tag_names = {tag['name'] for tag in bookmark['tags']}
        assert tag_names == {'Tag2', 'Tag3'}
        assert 'Tag1' not in tag_names
