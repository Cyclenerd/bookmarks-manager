import pytest
import os
import tempfile
from app import create_app
from config import Config


class TestConfig(Config):
    TESTING = True
    HTTP_AUTH_USERNAME = 'test'
    HTTP_AUTH_PASSWORD = 'test'
    FAVICON_CACHE_DIR = tempfile.mkdtemp()

    @staticmethod
    def get_database_path():
        return tempfile.mktemp(suffix='.db')


@pytest.fixture
def app():
    test_db_path = TestConfig.get_database_path()
    TestConfig.DATABASE_PATH = test_db_path

    app = create_app(TestConfig)

    yield app

    # Close any open database connections
    with app.app_context():
        from app.utils.database import close_db
        close_db()

    if os.path.exists(test_db_path):
        os.unlink(test_db_path)

    if os.path.exists(TestConfig.FAVICON_CACHE_DIR):
        import shutil
        shutil.rmtree(TestConfig.FAVICON_CACHE_DIR)


@pytest.fixture
def client(app):
    return app.test_client()


@pytest.fixture
def auth_headers():
    import base64
    credentials = base64.b64encode(b'test:test').decode('utf-8')
    return {'Authorization': f'Basic {credentials}'}