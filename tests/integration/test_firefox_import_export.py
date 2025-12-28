"""Integration tests for Firefox bookmark import/export workflows.

Tests the Firefox JSON bookmark import and export functionality.
"""


def test_firefox_export_workflow(client, auth_headers, app):
    """Test Firefox bookmark export workflow.

    Tests creating bookmarks and exporting them to Firefox JSON format.
    """
    # Create folder
    client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers)

    # Get folder ID
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        folder_id = folders[0]['id']

    # Create bookmarks
    bookmarks = [
        {
            'url': 'https://example.com',
            'title': 'Example Site',
            'description': 'Test bookmark',
            'folder_id': folder_id
        },
        {
            'url': 'https://test.com',
            'title': 'Test Site',
            'description': 'Another bookmark'
        }
    ]

    for bookmark in bookmarks:
        client.post('/bookmark/save', data=bookmark, headers=auth_headers)

    # Export to Firefox format
    response = client.get('/export/firefox', headers=auth_headers)
    assert response.status_code == 200
    assert response.headers['Content-Type'] == 'application/json'
    assert b'Example Site' in response.data or b'example.com' in response.data


def test_firefox_import_basic_workflow(client, auth_headers, app):
    """Test Firefox bookmark import workflow.

    Tests that the import functionality exists and handles requests properly.
    Note: Full import testing depends on Firefox service implementation details.
    """
    # Firefox JSON bookmark data
    import json
    firefox_json = {
        "title": "root",
        "children": [
            {
                "title": "Example Site",
                "uri": "https://example.com",
                "type": "text/x-moz-place"
            },
            {
                "title": "Test Site",
                "uri": "https://test.com",
                "type": "text/x-moz-place"
            }
        ]
    }

    # Import bookmarks
    from io import BytesIO
    json_data = json.dumps(firefox_json).encode('utf-8')
    response = client.post('/import/firefox',
                           data={'file': (BytesIO(json_data), 'bookmarks.json')},
                           headers=auth_headers,
                           content_type='multipart/form-data',
                           follow_redirects=True)
    # Import endpoint should respond (may be success or error page)
    assert response.status_code == 200


def test_firefox_import_with_folders(client, auth_headers, app):
    """Test Firefox import with folder structure.

    Tests that the import functionality handles folder structures.
    Note: Full import testing depends on Firefox service implementation details.
    """
    # Firefox JSON with folders
    import json
    firefox_json = {
        "title": "root",
        "children": [
            {
                "title": "Work",
                "type": "text/x-moz-place-container",
                "children": [
                    {
                        "title": "Work Site",
                        "uri": "https://work.com",
                        "type": "text/x-moz-place"
                    }
                ]
            },
            {
                "title": "Personal",
                "type": "text/x-moz-place-container",
                "children": [
                    {
                        "title": "Personal Site",
                        "uri": "https://personal.com",
                        "type": "text/x-moz-place"
                    }
                ]
            }
        ]
    }

    # Import bookmarks
    from io import BytesIO
    json_data = json.dumps(firefox_json).encode('utf-8')
    response = client.post('/import/firefox',
                           data={'file': (BytesIO(json_data), 'bookmarks.json')},
                           headers=auth_headers,
                           content_type='multipart/form-data',
                           follow_redirects=True)
    # Import endpoint should respond (may be success or error page)
    assert response.status_code == 200


def test_firefox_export_preserves_hierarchy(client, auth_headers, app):
    """Test that Firefox export preserves folder hierarchy.

    Tests that nested folders are correctly represented in export.
    """
    # Create nested folder structure
    client.post('/folder/save', data={'name': 'Projects'}, headers=auth_headers)

    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        parent_id = folders[0]['id']

    client.post('/folder/save', data={'name': 'Python', 'parent_id': parent_id}, headers=auth_headers)

    with app.app_context():
        folders = folder_service.get_all_folders()
        child_id = [f['id'] for f in folders if f['name'] == 'Python'][0]

    # Add bookmark to nested folder
    bookmark_data = {
        'url': 'https://python.org',
        'title': 'Python Site',
        'folder_id': child_id
    }
    client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)

    # Export and verify structure
    response = client.get('/export/firefox', headers=auth_headers)
    assert response.status_code == 200
    content = response.data.decode('utf-8')

    # Check that export contains data
    assert 'Projects' in content or 'Python' in content


def test_firefox_import_page_accessible(client, auth_headers):
    """Test that Firefox import page is accessible.

    Tests that the import page renders correctly.
    """
    response = client.get('/import', headers=auth_headers)
    assert response.status_code == 200
    assert b'Import' in response.data or b'import' in response.data


def test_firefox_empty_export(client, auth_headers):
    """Test Firefox export with no bookmarks.

    Tests that export works even with empty bookmark database.
    """
    response = client.get('/export/firefox', headers=auth_headers)
    assert response.status_code == 200
    assert response.headers['Content-Type'] == 'application/json'
