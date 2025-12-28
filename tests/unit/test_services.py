from app.services import bookmark_service, folder_service, tag_service


def test_folder_service_create(app):
    with app.app_context():
        folder_id = folder_service.create_folder('Test Folder')
        assert folder_id is not None

        folder = folder_service.get_folder(folder_id)
        assert folder['name'] == 'Test Folder'


def test_folder_service_update(app):
    with app.app_context():
        folder_id = folder_service.create_folder('Original Name')
        folder_service.update_folder(folder_id, 'Updated Name', None)

        folder = folder_service.get_folder(folder_id)
        assert folder['name'] == 'Updated Name'


def test_folder_service_delete(app):
    with app.app_context():
        folder_id = folder_service.create_folder('To Delete')
        folder_service.delete_folder(folder_id)

        folder = folder_service.get_folder(folder_id)
        assert folder is None


def test_tag_service_create(app):
    with app.app_context():
        tag_id = tag_service.create_tag('Test Tag')
        assert tag_id is not None

        tag = tag_service.get_tag(tag_id)
        assert tag['name'] == 'Test Tag'


def test_tag_service_update(app):
    with app.app_context():
        tag_id = tag_service.create_tag('Original Tag')
        tag_service.update_tag(tag_id, 'Updated Tag')

        tag = tag_service.get_tag(tag_id)
        assert tag['name'] == 'Updated Tag'


def test_tag_service_delete(app):
    with app.app_context():
        tag_id = tag_service.create_tag('To Delete')
        tag_service.delete_tag(tag_id)

        tag = tag_service.get_tag(tag_id)
        assert tag is None


def test_bookmark_service_create(app):
    with app.app_context():
        folder_id = folder_service.create_folder('Work')
        tag_id = tag_service.create_tag('Important')

        bookmark_id = bookmark_service.create_bookmark(
            'https://example.com',
            'Example',
            'Description',
            folder_id,
            [tag_id]
        )
        assert bookmark_id is not None

        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['title'] == 'Example'
        assert bookmark['url'] == 'https://example.com'
        assert len(bookmark['tags']) == 1


def test_bookmark_service_update(app):
    with app.app_context():
        bookmark_id = bookmark_service.create_bookmark(
            'https://example.com',
            'Original',
            'Description',
            None,
            []
        )

        bookmark_service.update_bookmark(
            bookmark_id,
            'https://example.com',
            'Updated',
            'New Description',
            None,
            []
        )

        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['title'] == 'Updated'
        assert bookmark['description'] == 'New Description'


def test_bookmark_service_delete(app):
    with app.app_context():
        bookmark_id = bookmark_service.create_bookmark(
            'https://example.com',
            'To Delete',
            'Description',
            None,
            []
        )

        bookmark_service.delete_bookmark(bookmark_id)
        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark is None


def test_bookmark_service_toggle_pin(app):
    with app.app_context():
        bookmark_id = bookmark_service.create_bookmark(
            'https://example.com',
            'Test',
            'Description',
            None,
            []
        )

        pinned = bookmark_service.toggle_pin(bookmark_id)
        assert pinned == 1

        pinned = bookmark_service.toggle_pin(bookmark_id)
        assert pinned == 0


def test_bookmark_service_get_all(app):
    with app.app_context():
        bookmark_service.create_bookmark('https://test1.com', 'Test 1', '', None, [])
        bookmark_service.create_bookmark('https://test2.com', 'Test 2', '', None, [])

        result = bookmark_service.get_all_bookmarks()
        assert result['total'] >= 2


def test_bookmark_service_search(app):
    with app.app_context():
        bookmark_service.create_bookmark('https://test.com', 'Searchable Title', 'Description', None, [])

        result = bookmark_service.get_all_bookmarks(search='Searchable')
        assert result['total'] >= 1
        assert any('Searchable' in b['title'] for b in result['bookmarks'])


def test_bookmark_service_filter_by_folder(app):
    with app.app_context():
        folder_id = folder_service.create_folder('Filter Test')
        bookmark_service.create_bookmark('https://test.com', 'Test', '', folder_id, [])

        result = bookmark_service.get_all_bookmarks(folder_id=folder_id)
        assert result['total'] >= 1


def test_bookmark_service_filter_by_tag(app):
    with app.app_context():
        tag_id = tag_service.create_tag('Filter Tag')
        bookmark_service.create_bookmark('https://test.com', 'Test', '', None, [tag_id])

        result = bookmark_service.get_all_bookmarks(tag_id=tag_id)
        assert result['total'] >= 1


def test_bookmark_service_sort(app):
    with app.app_context():
        bookmark_service.create_bookmark('https://b.com', 'B Title', '', None, [])
        bookmark_service.create_bookmark('https://a.com', 'A Title', '', None, [])

        result = bookmark_service.get_all_bookmarks(sort_by='title', sort_order='asc')
        titles = [b['title'] for b in result['bookmarks']]
        assert titles == sorted(titles)


def test_pinned_bookmarks_first(app):
    with app.app_context():
        bookmark_service.create_bookmark('https://test1.com', 'Normal', '', None, [])
        id2 = bookmark_service.create_bookmark('https://test2.com', 'Pinned', '', None, [])
        bookmark_service.toggle_pin(id2)

        result = bookmark_service.get_all_bookmarks()
        assert result['bookmarks'][0]['id'] == id2


def test_folder_hierarchy(app):
    """Test getting folder hierarchy"""
    with app.app_context():
        parent_id = folder_service.create_folder('Parent')
        folder_service.create_folder('Child', parent_id)

        hierarchy = folder_service.get_folder_hierarchy()
        assert len(hierarchy) >= 1

        parent_folder = next((f for f in hierarchy if f['id'] == parent_id), None)
        assert parent_folder is not None
        assert 'children' in parent_folder


def test_get_all_folders(app):
    """Test getting all folders"""
    with app.app_context():
        folder_service.create_folder('Folder 1')
        folder_service.create_folder('Folder 2')

        folders = folder_service.get_all_folders()
        assert len(folders) >= 2


def test_get_all_tags(app):
    """Test getting all tags"""
    with app.app_context():
        tag_service.create_tag('Tag 1')
        tag_service.create_tag('Tag 2')

        tags = tag_service.get_all_tags()
        assert len(tags) >= 2


def test_bookmark_with_multiple_tags(app):
    """Test creating bookmark with multiple tags"""
    with app.app_context():
        tag1 = tag_service.create_tag('Tag1')
        tag2 = tag_service.create_tag('Tag2')

        bookmark_id = bookmark_service.create_bookmark(
            'https://test.com',
            'Test',
            'Description',
            None,
            [tag1, tag2]
        )

        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark is not None


def test_nested_folder_deletion(app):
    """Test deleting folder with nested structure"""
    with app.app_context():
        parent_id = folder_service.create_folder('Parent')
        child_id = folder_service.create_folder('Child', parent_id)

        # Create bookmark in child folder
        bookmark_service.create_bookmark(
            'https://test.com',
            'Test',
            '',
            child_id,
            []
        )

        folder_service.delete_folder(parent_id)
        folder = folder_service.get_folder(parent_id)
        assert folder is None


def test_bookmark_search_partial_match(app):
    """Test searching bookmarks with partial matches"""
    with app.app_context():
        bookmark_service.create_bookmark(
            'https://github.com',
            'GitHub Repository',
            'Source code hosting',
            None,
            []
        )

        result = bookmark_service.get_all_bookmarks(search='repo')
        assert result['total'] >= 1
        assert any('GitHub' in b['title'] for b in result['bookmarks'])


def test_update_bookmark_with_new_tags(app):
    """Test updating bookmark with new tags"""
    with app.app_context():
        tag1 = tag_service.create_tag('OldTag')
        bookmark_id = bookmark_service.create_bookmark(
            'https://test.com',
            'Test',
            '',
            None,
            [tag1]
        )

        tag2 = tag_service.create_tag('NewTag')
        bookmark_service.update_bookmark(
            bookmark_id,
            'https://test.com',
            'Updated Test',
            '',
            None,
            [tag2]
        )

        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['title'] == 'Updated Test'


def test_folder_get_parent_chain_with_multiple_levels(app):
    """Test getting parent chain with multiple nested levels"""
    with app.app_context():
        folder1 = folder_service.create_folder('Level 1')
        folder2 = folder_service.create_folder('Level 2', folder1)
        folder3 = folder_service.create_folder('Level 3', folder2)

        folder = folder_service.get_folder(folder3)
        chain = folder['parent_chain']
        assert len(chain) == 2
        assert chain[0]['name'] == 'Level 1'
        assert chain[1]['name'] == 'Level 2'


def test_folder_get_parent_chain_with_none_parent(app):
    """Test getting parent chain when a parent is missing (edge case)"""
    with app.app_context():
        folder1 = folder_service.create_folder('Level 1')
        folder = folder_service.get_folder(folder1)
        chain = folder['parent_chain']
        assert len(chain) == 0


def test_folder_get_folder_with_descendants(app):
    """Test getting folder with all descendants"""
    with app.app_context():
        folder1 = folder_service.create_folder('Parent')
        folder2 = folder_service.create_folder('Child1', folder1)
        folder3 = folder_service.create_folder('Child2', folder1)
        folder4 = folder_service.create_folder('Grandchild', folder2)

        descendants = folder_service.get_folder_with_descendants(folder1)
        assert folder1 in descendants
        assert folder2 in descendants
        assert folder3 in descendants
        assert folder4 in descendants
        assert len(descendants) == 4


def test_bookmark_service_get_bookmark_not_found(app):
    """Test getting a non-existent bookmark"""
    with app.app_context():
        bookmark = bookmark_service.get_bookmark('non-existent-id')
        assert bookmark is None


def test_update_bookmark_with_favicon(app):
    """Test updating a bookmark with a favicon"""
    with app.app_context():
        bookmark_id = bookmark_service.create_bookmark('https://test.com', 'Test', 'Desc', None, [])

        # Update with favicon
        bookmark_service.update_bookmark(
            bookmark_id,
            'https://test.com',
            'Updated Test',
            'Updated Desc',
            None,
            [],
            'favicons/test.png'
        )

        bookmark = bookmark_service.get_bookmark(bookmark_id)
        assert bookmark['title'] == 'Updated Test'
        assert bookmark['favicon'] == 'favicons/test.png'


def test_folder_parent_chain_with_corrupted_parent(app):
    """Test folder parent chain when parent_id exists but parent record doesn't (edge case)"""
    with app.app_context():
        from app.utils.database import get_db
        import uuid

        # Create a folder with a non-existent parent_id
        folder_id = str(uuid.uuid4())
        nonexistent_parent_id = str(uuid.uuid4())

        db = get_db()
        db.execute(
            'INSERT INTO folders (id, name, parent_id) VALUES (?, ?, ?)',
            (folder_id, 'Test Folder', nonexistent_parent_id)
        )
        db.commit()

        # This should handle the missing parent gracefully
        folder = folder_service.get_folder(folder_id)
        assert folder is not None
        assert folder['name'] == 'Test Folder'
        assert len(folder['parent_chain']) == 0  # Should be empty since parent doesn't exist


def test_database_teardown(app):
    """Test database teardown function"""
    with app.app_context():
        from app.utils.database import teardown_db, get_db

        # Get a database connection
        get_db()

        # Call teardown
        teardown_db()

        # This should work without errors
        assert True
