"""Metadata service module.

This module provides functionality for fetching webpage metadata including
titles and other OpenGraph/meta tag information from URLs.
"""

import requests
from bs4 import BeautifulSoup


def fetch_page_metadata(url):
    """Fetch title and metadata from a web page URL.

    Attempts to extract the most appropriate title from:
    1. OpenGraph og:title meta tag
    2. Twitter twitter:title meta tag
    3. Standard HTML <title> tag

    Args:
        url (str): The URL to fetch metadata from

    Returns:
        dict: Dictionary containing:
            - title (str or None): Extracted page title
            - success (bool): Whether fetch succeeded
            - error (str): Error message if success is False

    Note:
        Follows redirects and uses a browser-like User-Agent.
        Has a 10 second timeout.
    """
    try:
        headers = {
            'User-Agent': ('Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 '
                           '(KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36')
        }
        response = requests.get(url, timeout=10, headers=headers, allow_redirects=True)
        response.raise_for_status()

        soup = BeautifulSoup(response.content, 'html.parser')

        title = extract_title(soup)

        return {
            'title': title,
            'success': True
        }
    except Exception as e:
        return {
            'title': None,
            'success': False,
            'error': str(e)
        }


def extract_title(soup):
    """Extract page title from parsed HTML.

    Tries multiple sources in order of preference:
    1. og:title meta property
    2. twitter:title meta name
    3. <title> tag

    Args:
        soup (BeautifulSoup): Parsed HTML document

    Returns:
        str or None: Extracted and trimmed title, or None if no title found
    """
    title = None

    og_title = soup.find('meta', property='og:title')
    if og_title and og_title.get('content'):
        title = og_title['content']

    if not title:
        twitter_title = soup.find('meta', attrs={'name': 'twitter:title'})
        if twitter_title and twitter_title.get('content'):
            title = twitter_title['content']

    if not title:
        title_tag = soup.find('title')
        if title_tag:
            title = title_tag.string

    return title.strip() if title else None
