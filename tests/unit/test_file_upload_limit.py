"""Test file upload size limit."""
from io import BytesIO


def test_max_content_length_configured(app):
    """Test that MAX_CONTENT_LENGTH is configured to 128 KB."""
    assert app.config['MAX_CONTENT_LENGTH'] == 128 * 1024


def test_import_firefox_file_too_large(client, auth_headers):
    """Test that files larger than 128 KB are rejected."""
    # Create a file larger than 128 KB
    large_content = b'{"test": "' + (b'x' * 130 * 1024) + b'"}'

    data = {
        'file': (BytesIO(large_content), 'large.json')
    }

    response = client.post('/import/firefox',
                           data=data,
                           content_type='multipart/form-data',
                           headers=auth_headers)

    # Flask returns 413 REQUEST ENTITY TOO LARGE for files exceeding MAX_CONTENT_LENGTH
    assert response.status_code == 413


def test_import_firefox_file_within_limit(client, auth_headers, monkeypatch):
    """Test that files within 128 KB limit are processed."""
    # Create a small valid Firefox JSON file
    small_content = b'{"guid": "root________", "title": "", "type": "text/x-moz-place-container", "children": []}'

    data = {
        'file': (BytesIO(small_content), 'small.json')
    }

    # Mock favicon service to avoid actual network calls
    def mock_download_favicon(url):
        return None

    import app.services.favicon_service
    monkeypatch.setattr(app.services.favicon_service, 'download_favicon', mock_download_favicon)

    response = client.post('/import/firefox',
                           data=data,
                           content_type='multipart/form-data',
                           headers=auth_headers)

    # Check that it's processed (either success or parse error, but not size error)
    assert response.status_code == 200
    assert b'File size exceeds 128 KB limit' not in response.data
