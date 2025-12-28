# Bookmarks Manager

[![Badge: Linux](https://img.shields.io/badge/Linux-FCC624.svg?logo=linux&logoColor=black)](#readme)
[![Badge: Terraform](https://img.shields.io/badge/Terraform-%235835CC.svg?logo=terraform&logoColor=white)](#readme)
[![Badge: Python](https://img.shields.io/badge/Python-3670A0?logo=python&logoColor=ffdd54)](#readme)
[![Badge: Docker](https://img.shields.io/badge/Docker-%230db7ed.svg?logo=docker&logoColor=white)](#readme)
[![Badge: Podman](https://img.shields.io/badge/Podman-%23892CA0.svg?logo=podman&logoColor=white)](#readme)
[![Badge: Kubernetes](https://img.shields.io/badge/Kubernetes-%23326ce5.svg?logo=kubernetes&logoColor=white)](#readme)
[![License: AGPL v3](https://img.shields.io/badge/License-AGPL%20v3-blue.svg)](https://www.gnu.org/licenses/agpl-3.0)

Bookmarks Manager is a lightweight, self-hosted bookmarking solution designed for speed and simplicity. Optimized for a single user, it features a simple SQLite backend and single user HTTP Basic Authentication, making it incredibly easy to deploy and maintain.

Take control of your web links with a powerful organization system, lightning-fast search, and a mobile-friendly interface.

## Features

### 📁 Organization
- **Hierarchical Folders**: Organize bookmarks into folders and subfolders (unlimited nesting)
- **Multi-level Tags**: Tag bookmarks with multiple tags for flexible organization
- **Pinned Bookmarks**: Pin important bookmarks to keep them at the top of any list

### 🔍 Search & Navigation
- **Live Search**: Real-time search with dropdown results as you type
- **Full-text Search**: Search across titles, URLs, descriptions, folder names, and tags
- **URL-based Navigation**: Direct access via URLs (e.g., `/?folder=<id>`, `/?tag=<id>`)
- **Sorting**: Sort by title, URL, or creation date (ascending/descending)

### ⌨️ Keyboard Shortcuts
- **`⌘+/` or `Ctrl+/`**: Focus search field (OS-aware)
- **`g` then `n`**: Add new bookmark (Gmail-style sequential shortcut)
- **`↑` / `↓`**: Navigate search results
- **`Enter`**: Open selected result or submit search

### 🎨 User Interface
- **Bootstrap 5 Dark Mode**: Modern, responsive design optimized for desktop and mobile
- **Automatic Favicon Fetching**: Fetches and caches website favicons automatically
- **Mobile-Optimized**: Responsive layout works great on phones and tablets
- **Fast Interaction**: Minimal JavaScript, form-based interactions

### 📤 Import/Export
- **Firefox Import**: Import bookmarks from Firefox JSON export files
- **Firefox Export**: Export all bookmarks to Firefox-compatible JSON format
- **Preserves Structure**: Maintains folder hierarchy and tags during import/export

### 🔒 Security
- **HTTP Basic Authentication**: Simple username/password protection
- **UUID-based IDs**: Uses UUIDs instead of sequential integers for all resources

### ⚡ Performance
- **SQLite Database**: Lightweight, serverless database
- **Favicon Caching**: Favicons cached locally to avoid repeated fetches
- **Optimized Queries**: Efficient database queries with proper indexing

## Getting Started

You can run the application locally or deploy it to a cloud platform like Google Cloud Platform.

The complete application is containerized and can be run with Docker, Podman or Kubernetes.
You can also run it locally with Python.
Deploying and running the application via a container is recommended.

**Clone the repository**:

```bash
git clone git@gitlab.com:Cyclenerd/bookmarks-manager.git
cd bookmarks-manager
```

### Local development

Prerequisites:

*   Python 3
*   pip
*   SQLite

1.  **Create and activate a virtual environment**:

    ```bash
    python3 -m venv .venv
    source .venv/bin/activate
    ```

2.  **Install the required dependencies**:

    ```bash
    pip3 install -r requirements.txt
    pip3 install -r requirements-dev.txt
    ```

3.  **Run the application**:

    ```bash
    python3 run.py
    ```

## Containerization with Docker or Podman

This project supports containerization with either [Docker](https://www.docker.com/products/docker-desktop) or [Podman](https://podman.io/). The instructions and scripts have been tested with Podman.

### Build the Container Image

Alternatively, you can use the `tools/build-podman.sh` script to automate this process with Podman.

```bash
docker build \
  --platform "linux/amd64" \
  -f Dockerfile -t bookmarks-manager .
```

(Optional) Export the container image to share and import:

```bash
docker save -o bookmarks-manager.tar bookmarks-manager
```

### Run the Container

To ensure your data persists, you need to mount the `./database` and `./app/static/favicons` directories from your host to the container.

```bash
docker run -d -p 8080:8080 \
  --platform "linux/amd64" \
  -v $(pwd)/database:/web/database:rw \
  -v $(pwd)/app/static/favicons:/web/app/static/favicons:rw \
  --name bookmarks-manager localhost/bookmarks-manager:latest
```

The application will then be accessible at `http://localhost:8080`.

### Docker Compose

Alternatively, you can use `docker-compose` to manage the application.

Then run:

```bash
docker-compose up -d
```

## Google Cloud Run Deployment

[![Run on Google Cloud](https://deploy.cloud.run/button.svg)](./gcp/README.md)

For more information, please see the [`gcp/README.md`](./gcp/README.md).

## Configuration

Configure the application using environment variables.
You can set them directly or use a `.env` file (recommended for local development).

### Authentication

The application is protected by HTTP Basic Auth. The default credentials are:

*   **Username**: `admin`
*   **Password**: `changeme`

You can change the credentials by setting the `HTTP_AUTH_USERNAME` and `HTTP_AUTH_PASSWORD` environment variables.

### Using .env File (recommended for local development)

```bash
# Copy the example file
cp .env.example .env

# Edit .env with your settings
nano .env
```

The `.env` file is automatically loaded by python-dotenv and is ignored by git for security.

### Required Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_AUTH_USERNAME` | Username for HTTP Basic Authentication | `admin` |
| `HTTP_AUTH_PASSWORD` | Password for HTTP Basic Authentication | `changeme` |

### Optional Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `SECRET_KEY` | Flask secret key for session security | generate with `secrets.token_hex(32)` |
| `DATABASE_PATH` | SQLite database file path | `database/bookmarks.db` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `FAVICON_CACHE_DIR` | Directory for favicon cache storage | `app/static/favicons` |
| `RATELIMIT_STORAGE_URI` | Rate limit storage backend (memory or Redis) | `memory://` |
| `RATELIMIT_DEFAULT` | Default rate limits for all endpoints | `100 per minute` |

### Rate Limiting

Flask-Limiter is configured to protect against abuse. Configure via environment variables:

- **In-memory storage** (default): `RATELIMIT_STORAGE_URI=memory://`
- **Redis storage**: `RATELIMIT_STORAGE_URI=redis://localhost:6379`
- **Custom limits**: `RATELIMIT_DEFAULT="100 per day, 20 per hour"`

### Example Configuration

#### Using .env File

```bash
# .env file
HTTP_AUTH_USERNAME=your-username
HTTP_AUTH_PASSWORD=your-secure-password
```

#### Using Environment Variables

```bash
export HTTP_AUTH_USERNAME="your-username"
export HTTP_AUTH_PASSWORD="your-secure-password"
export HTTP_PORT="8080"
```

## Usage

1. **Access**: Navigate to `http://localhost:8080`
2. **Login**: Use credentials (default: `admin/changeme`)
3. **Create Folders**: Organize with folders and subfolders
4. **Add Bookmarks**:
   - Click "Add Bookmark" or press `g` then `n`
   - Enter URL (title and favicon auto-fetched)
   - Add description, select folder, add tags
5. **Search**: Press `⌘+/` (Mac) or `Ctrl+/` (Windows/Linux) to search
6. **Navigate**: Use sidebar or URL parameters
7. **Pin**: Star important bookmarks to keep them at top
8. **Import/Export**:
   - Click "Import/Export" in the sidebar
   - **Import from Firefox**: Upload a Firefox JSON bookmark export
   - **Export to Firefox**: Download all bookmarks as Firefox JSON

## Documentation

All Python modules are documented with comprehensive docstrings.
View the documentation using Python's built-in pydoc:

```bash
# View module documentation
python -m pydoc app.services.bookmark_service

# Start an interactive documentation server
python -m pydoc -b
```

## License

This project is available under the terms of the [GNU Affero General Public License (AGPL)](LICENSE).

### Favicon

The favicon was generated using the following graphics from Twitter Twemoji:

*   **Graphics Title:** `1f516.svg`
*   **Graphics Author:** Copyright 2020 Twitter, Inc and other contributors ([https://github.com/twitter/twemoji](https://github.com/twitter/twemoji))
*   **Graphics Source:** [https://github.com/twitter/twemoji/blob/master/assets/svg/1f516.svg](https://github.com/twitter/twemoji/blob/master/assets/svg/1f516.svg)
*   **Graphics License:** CC-BY 4.0 ([https://creativecommons.org/licenses/by/4.0/](https://creativecommons.org/licenses/by/4.0/))
