def test_app_creation(app):
    assert app is not None
    assert app.config['TESTING'] is True


def test_database_initialization(app):
    with app.app_context():
        from app.utils.database import get_db
        db = get_db()
        cursor = db.execute("SELECT name FROM sqlite_master WHERE type='table'")
        tables = [row[0] for row in cursor.fetchall()]
        assert 'bookmarks' in tables
        assert 'folders' in tables
        assert 'tags' in tables
        assert 'bookmark_tags' in tables


def test_auth_required(client):
    response = client.get('/')
    assert response.status_code == 401


def test_auth_success(client, auth_headers):
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 200


def test_create_folder(client, auth_headers, app):
    response = client.post('/folder/save', data={'name': 'Test Folder'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_create_tag(client, auth_headers, app):
    response = client.post('/tag/save', data={'name': 'Test Tag'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_create_bookmark(client, auth_headers, app):
    client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Important'}, headers=auth_headers)

    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A test bookmark',
        'folder_id': 1,
        'tag_ids': [1]
    }

    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_get_bookmarks(client, auth_headers, app):
    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Test',
        'description': 'Test bookmark'
    }, headers=auth_headers)

    response = client.get('/', headers=auth_headers)
    assert response.status_code == 200
    assert b'Test' in response.data


def test_toggle_pin(client, auth_headers, app):
    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Test',
        'description': 'Test bookmark'
    }, headers=auth_headers)

    response = client.post('/bookmark/1/toggle-pin', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_update_bookmark(client, auth_headers, app):
    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Original Title',
        'description': 'Original description'
    }, headers=auth_headers)

    response = client.post('/bookmark/save', data={
        'bookmark_id': 1,
        'url': 'https://test.com',
        'title': 'Updated Title',
        'description': 'Updated description'
    }, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_delete_bookmark(client, auth_headers, app):
    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Test',
        'description': 'Test bookmark'
    }, headers=auth_headers)

    response = client.post('/bookmark/1/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_search_bookmarks(client, auth_headers, app):
    client.post('/bookmark/save', data={
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A searchable bookmark'
    }, headers=auth_headers)

    response = client.get('/search?q=searchable', headers=auth_headers)
    assert response.status_code == 200
    assert b'Example Site' in response.data
    assert b'Search Results' in response.data


def test_filter_by_folder(client, auth_headers, app):
    client.post('/folder/save', data={'name': 'Test Folder'}, headers=auth_headers)

    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Test',
        'folder_id': 1,
        'description': ''
    }, headers=auth_headers)

    response = client.get('/?folder=1', headers=auth_headers)
    assert response.status_code == 200
    assert b'Test' in response.data


def test_filter_by_tag(client, auth_headers, app):
    client.post('/tag/save', data={'name': 'Test Tag'}, headers=auth_headers)

    client.post('/bookmark/save', data={
        'url': 'https://test.com',
        'title': 'Test',
        'tag_ids': [1],
        'description': ''
    }, headers=auth_headers)

    response = client.get('/?tag=1', headers=auth_headers)
    assert response.status_code == 200


def test_update_folder(client, auth_headers, app):
    """Test updating a folder"""
    client.post('/folder/save', data={'name': 'Original Folder'}, headers=auth_headers)

    response = client.post('/folder/save', data={
        'folder_id': '1',
        'name': 'Updated Folder'
    }, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_delete_folder(client, auth_headers, app):
    """Test deleting a folder"""
    client.post('/folder/save', data={'name': 'Test Folder'}, headers=auth_headers)

    response = client.post('/folder/1/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_update_tag(client, auth_headers, app):
    """Test updating a tag"""
    client.post('/tag/save', data={'name': 'Original Tag'}, headers=auth_headers)

    response = client.post('/tag/save', data={
        'tag_id': '1',
        'name': 'Updated Tag'
    }, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_delete_tag(client, auth_headers, app):
    """Test deleting a tag"""
    client.post('/tag/save', data={'name': 'Test Tag'}, headers=auth_headers)

    response = client.post('/tag/1/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302


def test_live_search(client, auth_headers, app):
    """Test live search endpoint"""
    client.post('/bookmark/save', data={
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A test bookmark'
    }, headers=auth_headers)

    response = client.get('/api/search?q=example', headers=auth_headers)
    assert response.status_code == 200
    assert response.content_type == 'application/json'


def test_sort_bookmarks_by_title(client, auth_headers, app):
    """Test sorting bookmarks by title"""
    client.post('/bookmark/save', data={
        'url': 'https://z.com',
        'title': 'Z Site',
        'description': 'Last'
    }, headers=auth_headers)

    client.post('/bookmark/save', data={
        'url': 'https://a.com',
        'title': 'A Site',
        'description': 'First'
    }, headers=auth_headers)

    response = client.get('/?sort=title&order=asc', headers=auth_headers)
    assert response.status_code == 200


def test_sort_bookmarks_by_url(client, auth_headers, app):
    """Test sorting bookmarks by URL"""
    response = client.get('/?sort=url&order=desc', headers=auth_headers)
    assert response.status_code == 200


def test_search_redirect_from_index(client, auth_headers, app):
    """Test that search parameter on index page redirects to search page"""
    response = client.get('/?search=test', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302
    assert '/search?q=test' in response.location


def test_robots_txt(client):
    """Test robots.txt endpoint"""
    response = client.get('/robots.txt')
    assert response.status_code == 200
    assert response.mimetype == 'text/plain'


def test_favicon_ico(client):
    """Test favicon.ico endpoint"""
    response = client.get('/favicon.ico')
    assert response.status_code in [200, 404]


def test_fetch_metadata_api(client, auth_headers, app):
    """Test fetch metadata API endpoint"""
    response = client.post('/api/fetch-metadata',
                           json={'url': 'https://example.com'},
                           headers=auth_headers)
    assert response.status_code == 200
    assert response.content_type == 'application/json'


def test_fetch_metadata_api_no_url(client, auth_headers, app):
    """Test fetch metadata API without URL"""
    response = client.post('/api/fetch-metadata',
                           json={},
                           headers=auth_headers)
    assert response.status_code == 400
    json_data = response.get_json()
    assert json_data['success'] is False


def test_live_search_empty_query(client, auth_headers, app):
    """Test live search with empty query"""
    response = client.get('/api/search?q=', headers=auth_headers)
    assert response.status_code == 200
    json_data = response.get_json()
    assert 'bookmarks' in json_data
    assert json_data['bookmarks'] == []


def test_add_bookmark_form(client, auth_headers, app):
    """Test add bookmark form page"""
    response = client.get('/bookmark/add', headers=auth_headers)
    assert response.status_code == 200


def test_add_bookmark_form_with_folder(client, auth_headers, app):
    """Test add bookmark form with folder parameter"""
    client.post('/folder/save', data={'name': 'Test Folder'}, headers=auth_headers)
    response = client.get('/bookmark/add?folder=1', headers=auth_headers)
    assert response.status_code == 200


def test_add_bookmark_form_with_url_and_title(client, auth_headers, app):
    """Test add bookmark form with url and title parameters"""
    response = client.get('/bookmark/add?url=https://example.com&title=Example', headers=auth_headers)
    assert response.status_code == 200


def test_edit_bookmark_form(client, auth_headers, app):
    """Test edit bookmark form page"""
    with app.app_context():
        from app.services import bookmark_service
        bookmark_id = bookmark_service.create_bookmark('https://test.com', 'Test', 'Test bookmark', None, [])

    response = client.get(f'/bookmark/{bookmark_id}/edit', headers=auth_headers)
    assert response.status_code == 200


def test_edit_bookmark_form_not_found(client, auth_headers, app):
    """Test edit bookmark form with non-existent bookmark"""
    response = client.get('/bookmark/nonexistent-id/edit', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302
    assert response.location == '/'


def test_add_folder_form(client, auth_headers, app):
    """Test add folder form page"""
    response = client.get('/folder/add', headers=auth_headers)
    assert response.status_code == 200


def test_add_folder_form_with_parent(client, auth_headers, app):
    """Test add folder form with parent parameter"""
    client.post('/folder/save', data={'name': 'Parent Folder'}, headers=auth_headers)
    response = client.get('/folder/add?parent=1', headers=auth_headers)
    assert response.status_code == 200


def test_edit_folder_form(client, auth_headers, app):
    """Test edit folder form page"""
    with app.app_context():
        from app.services import folder_service
        folder_id = folder_service.create_folder('Test Folder')

    response = client.get(f'/folder/{folder_id}/edit', headers=auth_headers)
    assert response.status_code == 200


def test_edit_folder_form_not_found(client, auth_headers, app):
    """Test edit folder form with non-existent folder"""
    response = client.get('/folder/nonexistent-id/edit', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302
    assert response.location == '/'


def test_update_folder_circular_reference(client, auth_headers, app):
    """Test updating folder with circular reference"""
    with app.app_context():
        from app.services import folder_service
        parent_id = folder_service.create_folder('Parent Folder')
        child_id = folder_service.create_folder('Child Folder', parent_id)

    response = client.post('/folder/save', data={
        'folder_id': parent_id,
        'name': 'Parent Folder',
        'parent_id': child_id
    }, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 400


def test_add_tag_form(client, auth_headers, app):
    """Test add tag form page"""
    response = client.get('/tag/add', headers=auth_headers)
    assert response.status_code == 200


def test_edit_tag_form(client, auth_headers, app):
    """Test edit tag form page"""
    with app.app_context():
        from app.services import tag_service
        tag_id = tag_service.create_tag('Test Tag')

    response = client.get(f'/tag/{tag_id}/edit', headers=auth_headers)
    assert response.status_code == 200


def test_edit_tag_form_not_found(client, auth_headers, app):
    """Test edit tag form with non-existent tag"""
    response = client.get('/tag/nonexistent-id/edit', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302
    assert response.location == '/'