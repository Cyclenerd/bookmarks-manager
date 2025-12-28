"""Authentication utilities module.

This module provides HTTP Basic Authentication functionality for
protecting application routes and endpoints.
"""

from functools import wraps
from flask import request, Response, current_app
import secrets


def check_auth(username, password):
    """Verify username and password against configured credentials.

    Args:
        username (str): Provided username
        password (str): Provided password

    Returns:
        bool: True if credentials match configuration, False otherwise

    Note:
        Uses constant-time comparison to prevent timing attacks.
    """
    expected_username = current_app.config['HTTP_AUTH_USERNAME']
    expected_password = current_app.config['HTTP_AUTH_PASSWORD']

    username_match = secrets.compare_digest(username.encode('utf-8'), expected_username.encode('utf-8'))
    password_match = secrets.compare_digest(password.encode('utf-8'), expected_password.encode('utf-8'))

    return username_match and password_match


def authenticate():
    """Send HTTP 401 response requesting authentication.

    Returns:
        Response: Flask Response object with 401 status and WWW-Authenticate header
    """
    return Response(
        'Authentication required',
        401,
        {'WWW-Authenticate': 'Basic realm="Login Required"'}
    )


def requires_auth(f):
    """Decorator to require HTTP Basic Authentication for a route.

    Usage:
        @bp.route('/protected')
        @requires_auth
        def protected_view():
            return 'Secret content'

    Args:
        f (function): The view function to protect

    Returns:
        function: Decorated function that checks authentication before executing
    """
    @wraps(f)
    def decorated(*args, **kwargs):
        auth = request.authorization
        if not auth or not check_auth(auth.username, auth.password):
            return authenticate()
        return f(*args, **kwargs)
    return decorated
