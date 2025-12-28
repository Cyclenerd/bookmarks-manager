import time
from app.services import bookmark_service, folder_service, tag_service


def test_bookmark_timestamp_update(app):
    with app.app_context():
        # Create a bookmark
        bookmark_id = bookmark_service.create_bookmark(
            'https://example.com',
            'Original Title',
            'Description',
            None,
            []
        )
        bookmark = bookmark_service.get_bookmark(bookmark_id)
        created_at_initial = bookmark['created_at']

        # Wait a second to ensure timestamp will be different
        time.sleep(1.1)

        # Update the bookmark
        bookmark_service.update_bookmark(
            bookmark_id,
            'https://example.com',
            'Updated Title',
            'Description',
            None,
            []
        )

        bookmark_updated = bookmark_service.get_bookmark(bookmark_id)
        created_at_final = bookmark_updated['created_at']

        assert created_at_final > created_at_initial
        assert bookmark_updated['title'] == 'Updated Title'


def test_folder_timestamp_update(app):
    with app.app_context():
        folder_id = folder_service.create_folder('Original Folder')
        folder = folder_service.get_folder(folder_id)
        created_at_initial = folder['created_at']

        time.sleep(1.1)

        folder_service.update_folder(folder_id, 'Updated Folder', None)

        folder_updated = folder_service.get_folder(folder_id)
        created_at_final = folder_updated['created_at']

        assert created_at_final > created_at_initial


def test_tag_timestamp_update(app):
    with app.app_context():
        tag_id = tag_service.create_tag('Original Tag')
        tag = tag_service.get_tag(tag_id)
        created_at_initial = tag['created_at']

        time.sleep(1.1)

        tag_service.update_tag(tag_id, 'Updated Tag')

        tag_updated = tag_service.get_tag(tag_id)
        created_at_final = tag_updated['created_at']

        assert created_at_final > created_at_initial
