"""Configuration module for the bookmarks application.

This module defines the application configuration class that loads settings
from environment variables with sensible defaults.
"""

import os
import secrets
from dotenv import load_dotenv

load_dotenv()


class Config:
    """Application configuration class.

    Loads configuration from environment variables or provides defaults.
    All settings can be overridden via environment variables.

    Attributes:
        DATABASE_PATH: Path to SQLite database file
        DEBUG: Enable Flask debug mode
        FAVICON_CACHE_DIR: Directory for cached favicon images
        HTTP_AUTH_PASSWORD: Password for HTTP Basic Authentication
        HTTP_AUTH_USERNAME: Username for HTTP Basic Authentication
        HTTP_PORT: Port number for HTTP server
        MAX_CONTENT_LENGTH: Maximum file upload size in bytes
        PERMANENT_SESSION_LIFETIME: Session timeout in seconds
        RATELIMIT_DEFAULT: Default rate limit string (e.g., '100 per minute')
        RATELIMIT_STORAGE_URI: Storage backend URI for rate limiting
        SECRET_KEY: Secret key for session signing
        SESSION_COOKIE_HTTPONLY: Prevent JavaScript access to session cookie
        SESSION_COOKIE_SAMESITE: CSRF protection mode for cookies
        SESSION_COOKIE_SECURE: Require HTTPS for session cookies
        TESTING: Enable testing mode
    """
    DATABASE_PATH = os.environ.get('DATABASE_PATH', 'database/bookmarks.db')
    DEBUG = os.environ.get('DEBUG', True)
    FAVICON_CACHE_DIR = os.environ.get('FAVICON_CACHE_DIR', 'app/static/favicons')
    HTTP_AUTH_PASSWORD = os.environ.get('HTTP_AUTH_PASSWORD', 'changeme')
    HTTP_AUTH_USERNAME = os.environ.get('HTTP_AUTH_USERNAME', 'admin')
    HTTP_PORT = int(os.environ.get('HTTP_PORT', 8080))
    MAX_CONTENT_LENGTH = 128 * 1024  # 128 KB
    PERMANENT_SESSION_LIFETIME = 3600  # 1 hour
    RATELIMIT_DEFAULT = os.environ.get('RATELIMIT_DEFAULT', '100 per minute')
    RATELIMIT_STORAGE_URI = os.environ.get('RATELIMIT_STORAGE_URI', 'memory://')
    SECRET_KEY = os.environ.get('SECRET_KEY', secrets.token_hex(32))
    SESSION_COOKIE_HTTPONLY = True
    SESSION_COOKIE_SAMESITE = 'Lax'
    SESSION_COOKIE_SECURE = True
    TESTING = False
