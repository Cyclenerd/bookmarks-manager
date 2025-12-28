import pytest
import json
from app.services import firefox_service, bookmark_service, folder_service, tag_service
from app.utils.database import get_db


def test_export_to_firefox_json(app):
    """Test exporting bookmarks to Firefox JSON format."""
    with app.app_context():
        # Create test data
        folder_id = folder_service.create_folder('Test Folder', None)
        tag_id = tag_service.create_tag('test-tag')
        bookmark_service.create_bookmark(
            'https://example.com',
            'Example',
            'Test description',
            folder_id,
            [tag_id],
            None
        )

        # Export
        result = firefox_service.export_to_firefox_json()

        # Verify structure
        assert result['type'] == 'text/x-moz-place-container'
        assert result['guid'] == 'root________'
        assert len(result['children']) > 0

        # Find toolbar
        toolbar = result['children'][0]
        assert toolbar['guid'] == 'toolbar_____'
        assert toolbar['title'] == 'Bookmarks Toolbar'

        # Verify bookmark exists in export
        bookmarks_found = []

        def find_bookmarks(node):
            if node.get('type') == 'text/x-moz-place':
                bookmarks_found.append(node)
            for child in node.get('children', []):
                find_bookmarks(child)

        find_bookmarks(result)
        assert len(bookmarks_found) == 1
        assert bookmarks_found[0]['uri'] == 'https://example.com'
        assert bookmarks_found[0]['title'] == 'Example'


def test_import_from_firefox_json(app):
    """Test importing bookmarks from Firefox JSON format."""
    with app.app_context():
        # Create Firefox JSON structure
        firefox_data = {
            'guid': 'root________',
            'type': 'text/x-moz-place-container',
            'children': [{
                'guid': 'toolbar_____',
                'title': 'Bookmarks Toolbar',
                'type': 'text/x-moz-place-container',
                'children': [
                    {
                        'guid': 'folder1',
                        'title': 'Imported Folder',
                        'type': 'text/x-moz-place-container',
                        'children': [
                            {
                                'guid': 'bookmark1',
                                'title': 'Test Bookmark',
                                'type': 'text/x-moz-place',
                                'uri': 'https://test.com',
                                'tags': 'tag1,tag2',
                                'annos': [{
                                    'name': 'bookmarkProperties/description',
                                    'value': 'Test description'
                                }]
                            }
                        ]
                    }
                ]
            }]
        }

        # Import
        stats = firefox_service.import_from_firefox_json(firefox_data)

        # Verify stats
        assert stats['bookmarks'] == 1
        assert stats['folders'] == 1
        assert stats['tags'] == 2

        # Verify bookmark was created
        db = get_db()
        bookmark = db.execute('SELECT * FROM bookmarks WHERE url = ?', ('https://test.com',)).fetchone()
        assert bookmark is not None
        assert bookmark['title'] == 'Test Bookmark'
        assert bookmark['description'] == 'Test description'

        # Verify folder was created
        folder = db.execute('SELECT * FROM folders WHERE name = ?', ('Imported Folder',)).fetchone()
        assert folder is not None

        # Verify tags were created
        tag1 = db.execute('SELECT * FROM tags WHERE name = ?', ('tag1',)).fetchone()
        tag2 = db.execute('SELECT * FROM tags WHERE name = ?', ('tag2',)).fetchone()
        assert tag1 is not None
        assert tag2 is not None


def test_parse_firefox_json_valid(app):
    """Test parsing valid Firefox JSON."""
    with app.app_context():
        json_str = json.dumps({
            'guid': 'root________',
            'type': 'text/x-moz-place-container',
            'children': []
        })

        result = firefox_service.parse_firefox_json(json_str)
        assert result['guid'] == 'root________'


def test_parse_firefox_json_invalid(app):
    """Test parsing invalid Firefox JSON."""
    with app.app_context():
        with pytest.raises(ValueError, match='Invalid JSON format'):
            firefox_service.parse_firefox_json('not valid json')


def test_import_nested_folders(app):
    """Test importing nested folder structure."""
    with app.app_context():
        firefox_data = {
            'guid': 'root________',
            'type': 'text/x-moz-place-container',
            'children': [{
                'guid': 'toolbar_____',
                'title': 'Bookmarks Toolbar',
                'type': 'text/x-moz-place-container',
                'children': [
                    {
                        'guid': 'parent',
                        'title': 'Parent Folder',
                        'type': 'text/x-moz-place-container',
                        'children': [
                            {
                                'guid': 'child',
                                'title': 'Child Folder',
                                'type': 'text/x-moz-place-container',
                                'children': []
                            }
                        ]
                    }
                ]
            }]
        }

        stats = firefox_service.import_from_firefox_json(firefox_data)
        assert stats['folders'] == 2

        # Verify parent-child relationship
        db = get_db()
        parent = db.execute('SELECT * FROM folders WHERE name = ?', ('Parent Folder',)).fetchone()
        child = db.execute('SELECT * FROM folders WHERE name = ?', ('Child Folder',)).fetchone()

        assert parent is not None
        assert child is not None
        assert child['parent_id'] == parent['id']


def test_import_bookmark_without_folder(app):
    """Test importing bookmark without folder."""
    with app.app_context():
        firefox_data = {
            'guid': 'root________',
            'type': 'text/x-moz-place-container',
            'children': [{
                'guid': 'toolbar_____',
                'title': 'Bookmarks Toolbar',
                'type': 'text/x-moz-place-container',
                'children': [
                    {
                        'guid': 'bookmark1',
                        'title': 'Unfiled Bookmark',
                        'type': 'text/x-moz-place',
                        'uri': 'https://unfiled.com'
                    }
                ]
            }]
        }

        stats = firefox_service.import_from_firefox_json(firefox_data)
        assert stats['bookmarks'] == 1

        # Verify bookmark exists
        db = get_db()
        bookmark = db.execute('SELECT * FROM bookmarks WHERE url = ?', ('https://unfiled.com',)).fetchone()
        assert bookmark is not None
        assert bookmark['folder_id'] is None
