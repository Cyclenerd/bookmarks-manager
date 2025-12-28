"""Favicon service module.

This module handles downloading, processing, and caching of website favicons
for bookmark visual identification.
"""

import os
import requests
from urllib.parse import urlparse, urljoin
from flask import current_app
from io import BytesIO
from PIL import Image
from bs4 import BeautifulSoup


def download_favicon(url):
    """Download and cache a favicon for a URL.

    Attempts multiple strategies to find a favicon:
    1. HTML link tags with rel='icon', 'shortcut icon', or 'apple-touch-icon'
    2. Standard locations (/favicon.ico, /apple-touch-icon.png, /favicon.png)

    Args:
        url (str): The website URL to fetch favicon from

    Returns:
        str or None: Relative path to cached favicon (e.g., 'favicons/example.com.png'),
            or None if favicon could not be fetched
    """
    logger = current_app.logger
    logger.debug(f"Attempting to download favicon for: {url}")

    try:
        if not url or not isinstance(url, str):
            logger.error(f"Invalid URL provided: {url}")
            return None

        if not url.startswith(('http://', 'https://')):
            logger.error(f"URL missing http/https scheme: {url}")
            return None

        parsed_url = urlparse(url)
        domain = parsed_url.netloc
        base_url = f"{parsed_url.scheme}://{domain}"
        logger.debug(f"Parsed domain: {domain}, base_url: {base_url}")

        # Standard headers, but excluding 'br' (Brotli) to avoid garbled text
        # if the brotli library is not installed/working correctly.
        headers = {
            'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:144.0) Gecko/20100101 Firefox/144.0',
            'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8',
            'Accept-Language': 'en-US,en;q=0.8,de-DE;q=0.5,de;q=0.3',
            'Accept-Encoding': 'gzip, deflate',
            'DNT': '1',
            'Connection': 'keep-alive',
        }

        # Strategy 1: Parse HTML for link tags
        logger.debug(f"Strategy 1: Fetching HTML from {url}")
        try:
            response = requests.get(url, timeout=10, headers=headers, allow_redirects=True, verify=True)
            logger.debug(f"HTML fetch status: {response.status_code}")

            if response.status_code == 200:
                # Update domain and base_url if we were redirected
                if response.url != url:
                    logger.debug(f"Redirected from {url} to {response.url}")
                    parsed_final_url = urlparse(response.url)
                    domain = parsed_final_url.netloc
                    base_url = f"{parsed_final_url.scheme}://{domain}"
                    url = response.url  # Use final URL for relative path resolution

                content_type = response.headers.get('Content-Type', '').lower()
                if 'text/html' in content_type:
                    soup = BeautifulSoup(response.text, 'html.parser')

                    # Find all potential icon links
                    icon_links = []
                    for link in soup.find_all('link'):
                        rel = link.get('rel', [])
                        if isinstance(rel, str):
                            rel = [rel.lower()]
                        else:
                            rel = [r.lower() for r in rel]

                        # Check if any rel value contains 'icon' (e.g., 'icon', 'shortcut icon', 'apple-touch-icon')
                        # This handles both cases: rel=['icon'] and rel=['shortcut', 'icon']
                        if any('icon' in r for r in rel):
                            href = link.get('href')
                            if href:
                                icon_links.append(urljoin(url, href))

                    logger.debug(f"Found {len(icon_links)} icon link(s) in HTML: {icon_links}")

                    for icon_url in icon_links:
                        try:
                            logger.debug(f"Trying icon URL: {icon_url}")
                            icon_response = requests.get(icon_url, timeout=5, headers=headers,
                                                         allow_redirects=True, verify=True)
                            ct = icon_response.headers.get('Content-Type')
                            logger.debug(f"Icon fetch status: {icon_response.status_code}, Content-Type: {ct}")

                            if icon_response.status_code == 200:
                                ic_type = icon_response.headers.get('Content-Type', '').lower()
                                if 'image' in ic_type or 'octet-stream' in ic_type:
                                    res = save_favicon(icon_response.content, domain)
                                    if res:
                                        logger.info(f"Successfully downloaded favicon from HTML link for {domain}: {icon_url}")
                                        return res
                                    else:
                                        logger.error(f"Failed to save favicon from {icon_url}")
                                else:
                                    logger.warning(f"Skipping {icon_url}: not an image (Content-Type: {ic_type})")
                            else:
                                logger.error(f"Failed to fetch {icon_url}: HTTP {icon_response.status_code}")
                        except Exception as e:
                            logger.error(f"Exception fetching {icon_url}: {e}")
                            continue
                else:
                    logger.warning(f"Response is not HTML (Content-Type: {content_type})")
            else:
                logger.error(f"HTML fetch failed with status {response.status_code}")
        except Exception as e:
            logger.error(f"Strategy 1 failed: {e}")

        # Strategy 2: Check standard locations
        logger.debug(f"Strategy 2: Checking standard favicon locations for {domain}")
        favicon_urls = [
            f"{base_url}/favicon.ico",
            f"{base_url}/apple-touch-icon.png",
            f"{base_url}/favicon.png"
        ]

        for favicon_url in favicon_urls:
            try:
                logger.debug(f"Trying standard location: {favicon_url}")
                response = requests.get(favicon_url, timeout=5, headers=headers, allow_redirects=True, verify=True)
                ct = response.headers.get('Content-Type')
                logger.debug(f"Standard location fetch status: {response.status_code}, Content-Type: {ct}")

                if response.status_code == 200:
                    ic_type = response.headers.get('Content-Type', '').lower()
                    # Only accept if it's actually an image (sites often return HTML for 404s)
                    if 'image' in ic_type or 'octet-stream' in ic_type:
                        res = save_favicon(response.content, domain)
                        if res:
                            logger.info(f"Successfully downloaded favicon from standard location for {domain}: {favicon_url}")
                            return res
                        else:
                            logger.debug(f"Failed to save favicon from {favicon_url}")
                    else:
                        logger.debug(f"Skipping {favicon_url}: not an image (Content-Type: {ic_type})")
                else:
                    logger.debug(f"Standard location {favicon_url} returned HTTP {response.status_code}")
            except Exception as e:
                logger.error(f"Exception fetching {favicon_url}: {e}")
                continue

        logger.warning(f"Failed to download favicon for {domain}: no valid favicon found")
        return None
    except Exception as e:
        logger.error(f"Unexpected error downloading favicon for {url}: {e}", exc_info=True)
        return None


def save_favicon(content, domain):
    """Process and save favicon image to cache directory.

    Args:
        content (bytes): Raw image data
        domain (str): Domain name to use for cache filename

    Returns:
        str or None: Relative path to saved favicon, or None on error

    Note:
        Images are converted to RGBA and thumbnailed to 32x32 pixels.
        Domain name is sanitized (colons and slashes replaced with underscores).
    """
    logger = current_app.logger

    try:
        if len(content) > 2 * 1024 * 1024:
            logger.debug(f"Favicon for {domain} rejected: size {len(content)} bytes exceeds 2MB limit")
            return None

        img = Image.open(BytesIO(content))
        logger.debug(f"Favicon for {domain}: {img.format} image, size {img.width}x{img.height}")

        if img.width > 512 or img.height > 512:
            logger.debug(f"Favicon for {domain} rejected: dimensions {img.width}x{img.height} exceed 512x512 limit")
            return None

        img = img.convert('RGBA')
        img.thumbnail((32, 32), Image.Resampling.LANCZOS)

        favicon_dir = current_app.config['FAVICON_CACHE_DIR']
        os.makedirs(favicon_dir, exist_ok=True)

        filename = f"{domain.replace(':', '_').replace('/', '_')}.png"
        filepath = os.path.join(favicon_dir, filename)

        img.save(filepath, 'PNG')
        logger.debug(f"Saved favicon for {domain} to {filepath}")
        return f"favicons/{filename}"
    except Exception as e:
        logger.error(f"Failed to save favicon for {domain}: {e}")
        return None
