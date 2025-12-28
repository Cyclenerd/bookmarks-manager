"""Integration tests for folder workflows.

Tests the complete folder lifecycle including creation, hierarchical nesting,
editing, deletion, and folder-based bookmark filtering.
"""


def test_folder_creation_workflow(client, auth_headers, app):
    """Test folder creation workflow.

    Tests creating folders and verifying they appear in the hierarchy.
    """
    # Create root folder
    response = client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify folder exists
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        assert len(folders) == 1
        assert folders[0]['name'] == 'Work'
        assert folders[0]['parent_id'] is None


def test_nested_folder_creation_workflow(client, auth_headers, app):
    """Test nested folder creation workflow.

    Tests creating parent and child folders with hierarchical relationships.
    """
    # Create parent folder
    response = client.post('/folder/save', data={'name': 'Projects'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get parent folder ID
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        parent_id = folders[0]['id']

    # Create child folder
    response = client.post('/folder/save',
                           data={'name': 'Python Projects', 'parent_id': parent_id},
                           headers=auth_headers,
                           follow_redirects=False)
    assert response.status_code == 302

    # Verify hierarchy
    with app.app_context():
        hierarchy = folder_service.get_folder_hierarchy()
        assert len(hierarchy) == 1
        assert hierarchy[0]['name'] == 'Projects'
        assert len(hierarchy[0]['children']) == 1
        assert hierarchy[0]['children'][0]['name'] == 'Python Projects'


def test_deeply_nested_folder_hierarchy(client, auth_headers, app):
    """Test deeply nested folder hierarchy.

    Tests creating multiple levels of nested folders.
    """
    # Create folder hierarchy: Work > Projects > Python > Django
    response = client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers)
    assert response.status_code == 302

    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        work_id = folders[0]['id']

    response = client.post('/folder/save', data={'name': 'Projects', 'parent_id': work_id}, headers=auth_headers)
    assert response.status_code == 302

    with app.app_context():
        folders = folder_service.get_all_folders()
        projects_id = [f['id'] for f in folders if f['name'] == 'Projects'][0]

    response = client.post('/folder/save', data={'name': 'Python', 'parent_id': projects_id}, headers=auth_headers)
    assert response.status_code == 302

    with app.app_context():
        folders = folder_service.get_all_folders()
        python_id = [f['id'] for f in folders if f['name'] == 'Python'][0]

    response = client.post('/folder/save', data={'name': 'Django', 'parent_id': python_id}, headers=auth_headers)
    assert response.status_code == 302

    # Verify 4-level hierarchy
    with app.app_context():
        hierarchy = folder_service.get_folder_hierarchy()
        assert len(hierarchy) == 1
        assert hierarchy[0]['name'] == 'Work'
        assert hierarchy[0]['children'][0]['name'] == 'Projects'
        assert hierarchy[0]['children'][0]['children'][0]['name'] == 'Python'
        assert hierarchy[0]['children'][0]['children'][0]['children'][0]['name'] == 'Django'


def test_folder_edit_workflow(client, auth_headers, app):
    """Test folder editing workflow.

    Tests creating a folder, editing its name, and verifying changes.
    """
    # Create folder
    response = client.post('/folder/save', data={'name': 'Old Name'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get folder ID
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        folder_id = folders[0]['id']

    # Edit folder name
    response = client.post('/folder/save',
                           data={'folder_id': folder_id, 'name': 'New Name'},
                           headers=auth_headers,
                           follow_redirects=False)
    assert response.status_code == 302

    # Verify change
    with app.app_context():
        folder = folder_service.get_folder(folder_id)
        assert folder['name'] == 'New Name'


def test_folder_deletion_workflow(client, auth_headers, app):
    """Test folder deletion workflow.

    Tests creating and deleting a folder, ensuring bookmarks are handled.
    """
    # Create folder
    response = client.post('/folder/save', data={'name': 'To Delete'}, headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Get folder ID
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        folder_id = folders[0]['id']

    # Delete folder
    response = client.post(f'/folder/{folder_id}/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify deletion
    with app.app_context():
        folders = folder_service.get_all_folders()
        assert len(folders) == 0


def test_folder_with_bookmarks_deletion(client, auth_headers, app):
    """Test deleting folder with bookmarks.

    Tests that deleting a folder also affects its bookmarks (behavior depends on implementation).
    """
    # Create folder
    response = client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers)
    assert response.status_code == 302

    # Get folder ID
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        folder_id = folders[0]['id']

    # Create bookmark in folder
    bookmark_data = {
        'url': 'https://example.com',
        'title': 'Example',
        'folder_id': folder_id
    }
    response = client.post('/bookmark/save', data=bookmark_data, headers=auth_headers)
    assert response.status_code == 302

    # Delete folder
    response = client.post(f'/folder/{folder_id}/delete', headers=auth_headers, follow_redirects=False)
    assert response.status_code == 302

    # Verify bookmark still exists (folder might still be associated or null)
    with app.app_context():
        from app.services import bookmark_service
        bookmarks = bookmark_service.get_all_bookmarks()
        assert len(bookmarks['bookmarks']) == 1


def test_folder_filtering_workflow(client, auth_headers, app):
    """Test filtering bookmarks by folder.

    Tests creating folders and bookmarks, then filtering by folder.
    """
    # Create two folders
    client.post('/folder/save', data={'name': 'Work'}, headers=auth_headers)
    client.post('/folder/save', data={'name': 'Personal'}, headers=auth_headers)

    # Get folder IDs
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        work_id = [f['id'] for f in folders if f['name'] == 'Work'][0]
        personal_id = [f['id'] for f in folders if f['name'] == 'Personal'][0]

    # Create bookmarks in different folders
    client.post('/bookmark/save',
                data={'url': 'https://work.com', 'title': 'Work Site', 'folder_id': work_id},
                headers=auth_headers)
    client.post('/bookmark/save',
                data={'url': 'https://personal.com', 'title': 'Personal Site', 'folder_id': personal_id},
                headers=auth_headers)

    # Filter by work folder
    response = client.get(f'/?folder={work_id}', headers=auth_headers)
    assert response.status_code == 200
    assert b'Work Site' in response.data
    assert b'Personal Site' not in response.data

    # Filter by personal folder
    response = client.get(f'/?folder={personal_id}', headers=auth_headers)
    assert response.status_code == 200
    assert b'Personal Site' in response.data
    assert b'Work Site' not in response.data


def test_move_folder_parent_workflow(client, auth_headers, app):
    """Test moving folder to different parent.

    Tests changing the parent of a folder in the hierarchy.
    """
    # Create folder structure: Root1, Root2, Child
    client.post('/folder/save', data={'name': 'Root1'}, headers=auth_headers)
    client.post('/folder/save', data={'name': 'Root2'}, headers=auth_headers)

    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        root1_id = [f['id'] for f in folders if f['name'] == 'Root1'][0]
        root2_id = [f['id'] for f in folders if f['name'] == 'Root2'][0]

    # Create child under Root1
    client.post('/folder/save', data={'name': 'Child', 'parent_id': root1_id}, headers=auth_headers)

    with app.app_context():
        folders = folder_service.get_all_folders()
        child_id = [f['id'] for f in folders if f['name'] == 'Child'][0]

    # Move child to Root2
    response = client.post('/folder/save',
                           data={'folder_id': child_id, 'name': 'Child', 'parent_id': root2_id},
                           headers=auth_headers)
    assert response.status_code == 302

    # Verify hierarchy
    with app.app_context():
        hierarchy = folder_service.get_folder_hierarchy()
        root1 = [f for f in hierarchy if f['name'] == 'Root1'][0]
        root2 = [f for f in hierarchy if f['name'] == 'Root2'][0]
        assert len(root1['children']) == 0
        assert len(root2['children']) == 1
        assert root2['children'][0]['name'] == 'Child'
