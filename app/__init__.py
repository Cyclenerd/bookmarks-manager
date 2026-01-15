"""Flask application factory module.

This module provides the application factory function for creating and
configuring the Flask application instance with all necessary extensions,
blueprints, and database initialization.
"""

from flask import Flask, render_template
from werkzeug.exceptions import HTTPException
from flask_limiter import Limiter
from flask_limiter.util import get_remote_address
from config import Config
import os


def create_app(config_class=Config):
    """Create and configure the Flask application.

    This factory function creates a new Flask application instance, configures
    it with the provided configuration class, initializes rate limiting, registers
    blueprints, and sets up the database.

    Args:
        config_class: Configuration class to use (default: Config from config.py)

    Returns:
        Flask: Configured Flask application instance

    Example:
        >>> app = create_app()
        >>> app.run()
    """
    app = Flask(__name__)
    app.config.from_object(config_class)

    os.makedirs(app.config['FAVICON_CACHE_DIR'], exist_ok=True)

    app.limiter = Limiter(
        get_remote_address,
        app=app,
        storage_uri=app.config['RATELIMIT_STORAGE_URI'],
        default_limits=[app.config['RATELIMIT_DEFAULT']]
    )

    @app.after_request
    def add_security_headers(response):
        """Add security headers to all responses."""
        response.headers['X-Content-Type-Options'] = 'nosniff'
        response.headers['X-Frame-Options'] = 'DENY'
        response.headers['X-XSS-Protection'] = '1; mode=block'
        response.headers['Referrer-Policy'] = 'strict-origin-when-cross-origin'
        response.headers['Strict-Transport-Security'] = 'max-age=31536000; includeSubDomains'
        csp_policy = (
            "default-src 'self'; "
            "script-src 'self' 'unsafe-inline'; "
            "style-src 'self' 'unsafe-inline'; "
            "img-src 'self' data:; "
            "font-src 'self'; "
            "connect-src 'self'; "
            "frame-ancestors 'none'; "
            "base-uri 'self'; "
            "form-action 'self'"
        )
        response.headers['Content-Security-Policy'] = csp_policy
        return response

    @app.errorhandler(HTTPException)
    def handle_exception(e):
        """Render error template for HTTP errors."""
        return render_template('error.html',
                               error_code=e.code,
                               error_name=e.name,
                               error_description=e.description), e.code

    from app.routes import main
    app.register_blueprint(main.bp)

    from app.utils.database import init_db, close_db
    with app.app_context():
        init_db()

    app.teardown_appcontext(close_db)

    return app
