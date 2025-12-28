import os
from unittest.mock import Mock, patch
from io import BytesIO
from PIL import Image
from app.services.favicon_service import download_favicon, save_favicon


class TestFaviconService:
    """Test favicon downloading and saving"""

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_success(self, mock_get, app):
        """Test successful favicon download"""
        with app.app_context():
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.url = 'https://example.com'
            mock_response.content = b'fake_favicon_data'
            mock_response.headers = {'Content-Type': 'image/png'}
            mock_get.return_value = mock_response

            # Create a fake image to save
            with patch('app.services.favicon_service.save_favicon', return_value='favicons/test.png'):
                result = download_favicon('https://example.com')
                assert result == 'favicons/test.png'

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_failure(self, mock_get, app):
        """Test favicon download failure"""
        with app.app_context():
            mock_get.side_effect = Exception("Connection error")
            result = download_favicon('https://example.com')
            assert result is None

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_404(self, mock_get, app):
        """Test favicon download with 404 response"""
        with app.app_context():
            mock_response = Mock()
            mock_response.status_code = 404
            mock_get.return_value = mock_response

            result = download_favicon('https://example.com')
            assert result is None

    def test_save_favicon_success(self, app):
        """Test saving a valid favicon"""
        with app.app_context():
            # Create a simple test image
            img = Image.new('RGB', (100, 100), color='red')
            img_bytes = BytesIO()
            img.save(img_bytes, format='PNG')
            img_bytes.seek(0)

            result = save_favicon(img_bytes.getvalue(), 'example.com')
            assert result is not None
            assert 'favicons/' in result
            assert 'example.com.png' in result

    def test_save_favicon_invalid_image(self, app):
        """Test saving invalid image data"""
        with app.app_context():
            result = save_favicon(b'not_an_image', 'example.com')
            assert result is None

    def test_save_favicon_sanitizes_domain(self, app):
        """Test that domain names are sanitized"""
        with app.app_context():
            img = Image.new('RGB', (100, 100), color='blue')
            img_bytes = BytesIO()
            img.save(img_bytes, format='PNG')
            img_bytes.seek(0)

            result = save_favicon(img_bytes.getvalue(), 'example.com:8080/path')
            assert result is not None
            assert ':' not in result
            assert '/' not in result.split('/')[-1]

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_malformed_url(self, mock_get, app):
        """Test favicon download with malformed URL"""
        with app.app_context():
            result = download_favicon('not-a-valid-url')
            assert result is None

    def test_download_favicon_with_exception_in_parsing(self, app):
        """Test favicon download when URL parsing fails"""
        with app.app_context():
            # Pass None or an invalid type that will cause urlparse to fail
            with patch('app.services.favicon_service.urlparse', side_effect=Exception("Parse error")):
                result = download_favicon('https://example.com')
                assert result is None

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_from_html(self, mock_get, app):
        """Test favicon download discovered from HTML <link> tags"""
        with app.app_context():
            # Mock responses for different calls
            def side_effect(url, **kwargs):
                mock_resp = Mock()
                if url.endswith('favicon.ico') or url.endswith('apple-touch-icon.png') or url.endswith('favicon.png'):
                    mock_resp.status_code = 404
                    mock_resp.url = url
                elif url == 'https://example.com':
                    mock_resp.status_code = 200
                    mock_resp.url = url
                    mock_resp.text = '<html><head><link rel="shortcut icon" href="/path/to/custom.ico"></head></html>'
                    mock_resp.headers = {'Content-Type': 'text/html'}
                elif url == 'https://example.com/path/to/custom.ico':
                    mock_resp.status_code = 200
                    mock_resp.url = url
                    mock_resp.content = b'custom_favicon_data'
                    mock_resp.headers = {'Content-Type': 'image/png'}
                else:
                    mock_resp.status_code = 404
                    mock_resp.url = url
                return mock_resp

            mock_get.side_effect = side_effect

            with patch('app.services.favicon_service.save_favicon', return_value='favicons/custom.png'):
                result = download_favicon('https://example.com')
                assert result == 'favicons/custom.png'

    @patch('app.services.favicon_service.requests.get')
    def test_download_favicon_with_redirect(self, mock_get, app):
        """Test favicon download with a redirect and domain update"""
        with app.app_context():
            def side_effect(url, **kwargs):
                mock_resp = Mock()
                if url == 'http://old.com':
                    mock_resp.status_code = 200
                    mock_resp.url = 'https://new.com/login'
                    mock_resp.text = '<html><head></head><body>No icon tags here</body></html>'
                    mock_resp.headers = {'Content-Type': 'text/html'}
                elif url == 'https://new.com/favicon.ico':
                    mock_resp.status_code = 200
                    mock_resp.url = 'https://new.com/favicon.ico'
                    mock_resp.content = b'fake_favicon_data'
                    mock_resp.headers = {'Content-Type': 'image/x-icon'}
                else:
                    mock_resp.status_code = 404
                return mock_resp

            mock_get.side_effect = side_effect

            with patch('app.services.favicon_service.save_favicon', return_value='favicons/new.png'):
                result = download_favicon('http://old.com')
                assert result == 'favicons/new.png'

                # Verify that Strategy 2 searched on new.com, not old.com
                found_new_com_search = False
                for call in mock_get.call_args_list:
                    if 'new.com/favicon.ico' in call.args[0]:
                        found_new_com_search = True
                        break
                assert found_new_com_search, "Strategy 2 should have searched on new.com"

    def test_download_favicon_real_nkn_it(self, app):
        """Test real favicon download for nkn-it.de"""
        with app.app_context():
            # This is a real network request
            result = download_favicon('https://www.nkn-it.de/')
            assert result is not None
            assert 'favicons/www.nkn-it.de.png' in result

            # Verify the file exists
            favicon_dir = app.config['FAVICON_CACHE_DIR']
            filepath = os.path.join(favicon_dir, 'www.nkn-it.de.png')
            assert os.path.exists(filepath)
