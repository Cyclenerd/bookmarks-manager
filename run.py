#!/usr/bin/env python3
"""Application entry point for the bookmarks manager.

This script creates and runs the Flask application with the configured
HTTP port and host settings.
"""

from app import create_app
from config import Config

app = create_app()

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=Config.HTTP_PORT)
