"""Unit tests for Flask-Limiter rate limiting functionality.

This module contains tests to verify that Flask-Limiter is properly configured
and enforcing the default rate limit of 100 requests per minute.
"""


def test_limiter_is_configured(app):
    """Test that Flask-Limiter is properly configured on the app.

    Args:
        app: Flask application fixture

    Verifies that the limiter is attached to the app instance
    and has the correct default limit configured.
    """
    assert hasattr(app, 'limiter')
    assert app.limiter is not None
    assert app.config['RATELIMIT_DEFAULT'] == '100 per minute'


def test_rate_limit_allows_normal_requests(client, auth_headers):
    """Test that normal usage within limits is allowed.

    Args:
        client: Flask test client fixture
        auth_headers: HTTP Basic Auth headers fixture

    Makes 10 requests which should all succeed since it's well below
    the 100 requests per minute limit.
    """
    for i in range(10):
        response = client.get('/', headers=auth_headers)
        assert response.status_code == 200, f"Request {i + 1} failed"


def test_rate_limit_enforced_at_100_per_minute(client, auth_headers):
    """Test that rate limit is enforced at 100 requests per minute.

    Args:
        client: Flask test client fixture
        auth_headers: HTTP Basic Auth headers fixture

    Makes 101 requests rapidly to verify:
    1. First 100 requests succeed
    2. 101st request is rate limited (429 status code)
    """
    # Make 100 requests - all should succeed
    for i in range(100):
        response = client.get('/', headers=auth_headers)
        assert response.status_code == 200, f"Request {i + 1} was rate limited unexpectedly"

    # 101st request should be rate limited
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 429, "Rate limit was not enforced at 101st request"


def test_rate_limit_returns_429_when_exceeded(client, auth_headers):
    """Test that rate limit returns 429 status when exceeded.

    Args:
        client: Flask test client fixture
        auth_headers: HTTP Basic Auth headers fixture

    Verifies that Flask-Limiter properly returns HTTP 429 when
    the rate limit is exceeded.
    """
    # Exhaust the rate limit with 100 requests
    for i in range(100):
        client.get('/', headers=auth_headers)

    # Next request should return 429
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 429


def test_rate_limit_applies_to_single_endpoint(client, auth_headers):
    """Test that rate limiting is enforced on a single endpoint.

    Args:
        client: Flask test client fixture
        auth_headers: HTTP Basic Auth headers fixture

    Verifies that making 101 requests to the same endpoint triggers
    the rate limit on the 101st request.
    """
    # Make 100 requests to the same endpoint
    for i in range(100):
        response = client.get('/', headers=auth_headers)
        assert response.status_code == 200, f"Request {i + 1} failed unexpectedly"

    # 101st request should be rate limited
    response = client.get('/', headers=auth_headers)
    assert response.status_code == 429, "Rate limit not enforced at 101st request"


def test_rate_limit_storage_configured(app):
    """Test that rate limit storage backend is configured.

    Args:
        app: Flask application fixture

    Verifies that the storage URI is set (memory:// for testing).
    """
    assert 'RATELIMIT_STORAGE_URI' in app.config
    assert app.config['RATELIMIT_STORAGE_URI'] == 'memory://'
