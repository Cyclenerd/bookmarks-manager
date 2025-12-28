import requests
from unittest.mock import Mock, patch
from bs4 import BeautifulSoup
from app.services.metadata_service import (
    fetch_page_metadata,
    extract_title
)


class TestMetadataService:
    """Test metadata extraction from web pages"""

    def test_extract_title_from_og_tag(self):
        """Test extracting title from Open Graph meta tag"""
        html = '<html><head><meta property="og:title" content="OG Title"></head></html>'
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title == "OG Title"

    def test_extract_title_from_twitter_tag(self):
        """Test extracting title from Twitter meta tag"""
        html = '<html><head><meta name="twitter:title" content="Twitter Title"></head></html>'
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title == "Twitter Title"

    def test_extract_title_from_title_tag(self):
        """Test extracting title from title tag"""
        html = '<html><head><title>Page Title</title></head></html>'
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title == "Page Title"

    def test_extract_title_prefers_og_over_twitter(self):
        """Test that OG title is preferred over Twitter title"""
        html = '''<html><head>
            <meta property="og:title" content="OG Title">
            <meta name="twitter:title" content="Twitter Title">
        </head></html>'''
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title == "OG Title"

    def test_extract_title_strips_whitespace(self):
        """Test that title whitespace is stripped"""
        html = '<html><head><title>  Spaced Title  </title></head></html>'
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title == "Spaced Title"

    def test_extract_title_returns_none_when_missing(self):
        """Test that None is returned when no title found"""
        html = '<html><head></head></html>'
        soup = BeautifulSoup(html, 'html.parser')
        title = extract_title(soup)
        assert title is None

    @patch('app.services.metadata_service.requests.get')
    def test_fetch_page_metadata_success(self, mock_get):
        """Test successful metadata fetch"""
        html = '<html><head><title>Test Page</title></head></html>'
        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.content = html.encode()
        mock_response.url = 'https://example.com'
        mock_get.return_value = mock_response

        result = fetch_page_metadata('https://example.com')
        assert result['success'] is True
        assert result['title'] == 'Test Page'

    @patch('app.services.metadata_service.requests.get')
    def test_fetch_page_metadata_failure(self, mock_get):
        """Test metadata fetch failure"""
        mock_get.side_effect = Exception("Connection error")

        result = fetch_page_metadata('https://example.com')
        assert result['success'] is False
        assert 'error' in result
        assert result['title'] is None

    @patch('app.services.metadata_service.requests.get')
    def test_fetch_page_metadata_timeout(self, mock_get):
        """Test metadata fetch with timeout"""
        mock_get.side_effect = requests.exceptions.Timeout("Timeout")

        result = fetch_page_metadata('https://example.com')
        assert result['success'] is False
        assert 'Timeout' in result['error']
