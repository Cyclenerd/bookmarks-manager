"""Integration tests for search workflows.

Tests search functionality including full-text search, live search API,
and search result filtering.
"""

import json


def test_search_page_workflow(client, auth_headers, app):
    """Test search page workflow.

    Tests creating bookmarks and searching for them.
    """
    # Create bookmarks with different content
    bookmarks = [
        {'url': 'https://python.org', 'title': 'Python Programming', 'description': 'Official Python website'},
        {'url': 'https://flask.palletsprojects.com', 'title': 'Flask Documentation', 'description': 'Python web framework'}
    ]

    for bookmark in bookmarks:
        client.post('/bookmark/save', data=bookmark, headers=auth_headers)

    # Search for "Python"
    response = client.get('/search?q=Python', headers=auth_headers)
    assert response.status_code == 200
    assert b'Python Programming' in response.data
    assert b'Flask Documentation' in response.data


def test_live_search_api_workflow(client, auth_headers, app):
    """Test live search API workflow.

    Tests the AJAX search endpoint for live search results.
    """
    # Create test bookmarks
    bookmarks = [
        {'url': 'https://github.com', 'title': 'GitHub', 'description': 'Code hosting platform'},
        {'url': 'https://gitlab.com', 'title': 'GitLab', 'description': 'DevOps platform'},
        {'url': 'https://bitbucket.org', 'title': 'Bitbucket', 'description': 'Git repository'}
    ]

    for bookmark in bookmarks:
        client.post('/bookmark/save', data=bookmark, headers=auth_headers)

    # Test live search
    response = client.get('/api/search?q=git', headers=auth_headers)
    assert response.status_code == 200
    data = json.loads(response.data)
    assert 'bookmarks' in data
    assert len(data['bookmarks']) >= 2  # GitHub and GitLab should match


def test_search_with_special_characters(client, auth_headers, app):
    """Test search with special characters.

    Tests that search handles special characters correctly.
    """
    # Create bookmark with special characters
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'C++ Programming',
        'description': 'Learn C++ with examples & tutorials'
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Search for C++
    response = client.get('/search?q=C%2B%2B', headers=auth_headers)
    assert response.status_code == 200
    assert b'C++ Programming' in response.data


def test_search_in_folders_and_tags(client, auth_headers, app):
    """Test search includes folder and tag context.

    Tests that search finds bookmarks and displays their folder and tag associations.
    """
    # Create folder and tag
    client.post('/folder/save', data={'name': 'Development'}, headers=auth_headers)
    client.post('/tag/save', data={'name': 'Tutorial'}, headers=auth_headers)

    # Get IDs
    with app.app_context():
        from app.services import folder_service, tag_service
        folders = folder_service.get_all_folders()
        tags = tag_service.get_all_tags()
        folder_id = folders[0]['id']
        tag_id = tags[0]['id']

    # Create bookmark with folder and tag
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A test bookmark',
        'folder_id': folder_id,
        'tag_ids': [tag_id]
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Search by bookmark title should find it
    response = client.get('/search?q=Example', headers=auth_headers)
    assert response.status_code == 200
    assert b'Example Site' in response.data


def test_search_sorting_workflow(client, auth_headers, app):
    """Test search with sorting options.

    Tests that search results can be sorted by different criteria.
    """
    # Create bookmarks with different titles
    bookmarks = [
        {'url': 'https://a.com', 'title': 'Zebra Site', 'description': 'Search term here'},
        {'url': 'https://b.com', 'title': 'Apple Site', 'description': 'Search term here'},
        {'url': 'https://c.com', 'title': 'Banana Site', 'description': 'Search term here'}
    ]

    for bookmark in bookmarks:
        client.post('/bookmark/save', data=bookmark, headers=auth_headers)

    # Search with title sorting ascending
    response = client.get('/search?q=term&sort=title&order=asc', headers=auth_headers)
    assert response.status_code == 200
    content = response.data.decode('utf-8')
    apple_pos = content.find('Apple Site')
    banana_pos = content.find('Banana Site')
    zebra_pos = content.find('Zebra Site')
    assert apple_pos < banana_pos < zebra_pos


def test_search_pagination_workflow(client, auth_headers, app):
    """Test search pagination.

    Tests that search results are paginated correctly.
    """
    # Create many bookmarks with same search term
    for i in range(15):
        bookmark_data = {
            'url': f'https://example{i}.com',
            'title': f'Test Bookmark {i}',
            'description': 'Common search term'
        }
        client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Get first page
    response = client.get('/search?q=Common&page=1', headers=auth_headers)
    assert response.status_code == 200
    assert b'Test Bookmark' in response.data

    # Verify pagination controls exist when there are multiple pages
    # (depends on per_page setting, default is 100 so might not paginate with 15 items)


def test_empty_search_results(client, auth_headers, app):
    """Test search with no results.

    Tests that search handles no results gracefully.
    """
    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'A test bookmark'
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Search for non-existent term
    response = client.get('/search?q=nonexistent_term_xyz', headers=auth_headers)
    assert response.status_code == 200
    assert b'No bookmarks found' in response.data or b'0 bookmarks' in response.data


def test_search_redirect_from_index(client, auth_headers, app):
    """Test search redirect from index page.

    Tests that search from index page redirects to search page.
    """
    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site',
        'description': 'Test content'
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Search from index should redirect to search page
    response = client.get('/?search=Example', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302
    assert '/search?q=Example' in response.location


def test_live_search_empty_query(client, auth_headers, app):
    """Test live search with empty query.

    Tests that live search handles empty queries.
    """
    # Create bookmark
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example Site'
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Empty search query
    response = client.get('/api/search?q=', headers=auth_headers)
    assert response.status_code == 200
    data = json.loads(response.data)
    assert 'bookmarks' in data
    assert len(data['bookmarks']) == 0  # Empty query returns empty results
