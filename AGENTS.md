# AGENTS.md

This file provides guidance to AI agents when working with code in this repository.

## System Architecture

This document provides a high-level overview of the `bookmarks` project, a self-hosted bookmark manager.

## Project Overview

**Bookmarks Manager** is a single-user web application for managing bookmarks with:
- Hierarchical folder organization (unlimited nesting)
- Multi-tag support
- Full-text search with live results
- Keyboard shortcuts for power users
- Mobile-optimized responsive UI
- HTTP Basic Authentication

## Components

### 1. Flask Application (`app/`)
The core logic resides in a Flask web application.

**Architecture Pattern**: Service-oriented architecture
- **Routes** (`app/routes/`): Handle HTTP requests and responses
- **Services** (`app/services/`): Business logic and database operations
- **Utils** (`app/utils/`): Shared utilities (database, favicon fetching)
- **Templates** (`app/templates/`): Jinja2 HTML templates
- **Static** (`app/static/`): CSS, favicon cache

### 2. Database Layer
- **Engine**: SQLite (serverless, file-based)
- **Schema**: UUIDs for all primary keys
- **Tables**: bookmarks, folders, tags, bookmark_tags
- **Features**: Hierarchical folders (self-referencing), many-to-many tags

### 3. Frontend Layer
- **Framework**: Bootstrap 5 (Dark Mode)
- **Approach**: HTML forms with minimal JavaScript
- **JavaScript**: Only for live search and keyboard shortcuts
- **Responsive**: Mobile-first design

## Directory Structure

```
bookmarks/
├── app/
│   ├── __init__.py          # Flask app initialization
│   ├── routes/              # Route handlers
│   │   └── main.py          # Main routes
│   ├── services/            # Business logic
│   │   ├── bookmark_service.py
│   │   ├── folder_service.py
│   │   └── tag_service.py
│   ├── utils/               # Utility functions
│   │   ├── db.py            # Database utilities
│   │   └── favicon.py       # Favicon fetching
│   ├── templates/           # Jinja2 templates
│   └── static/              # Static files & favicon cache
├── tests/                   # Pytest test suite
│   └── unit/
│       └── test_app.py
├── config.py                # Configuration
├── run.py                   # Application entry point
└── requirements.txt         # Python dependencies
```

## Technologies Used

*   **Backend:** Python 3, Flask, SQLite
*   **Frontend:** HTML5, Jinja2, JavaScript (ES6), Bootstrap 5 (Dark Mode)
*   **HTTP Server:** Flask development server (Werkzeug)
*   **Authentication:** HTTP Basic Auth
*   **Image Processing:** Pillow (favicon processing)
*   **HTTP Client:** Requests (favicon fetching)

## Python Coding Style

Follow these coding style rules when writing Python code:

*   **Linter:** Code must pass `flake8 --ignore=W292,W503 --max-line-length=127 --show-source --statistics *.py app/*.py app/routes/*.py app/services/*.py tests/*.py tests/integration/*.py tests/unit/*.py`
*   **Line Length:** Maximum line length is 127 characters
*   **Blank Lines:** No blank line should contain whitespace (trailing whitespace is not allowed)
*   **End of File:** W292 is ignored (no blank line required at end of file)
*   **Binary Operator:** W503 for line break before binary operator is ignored
*   **Spaces:** Indent with spaces

### Documentation Standards

*   **Module Docstrings:** All Python modules must have a module-level docstring describing their purpose
*   **Function Docstrings:** All functions must have docstrings with Google-style formatting:
    - Brief description
    - Args: Parameter names, types, and descriptions
    - Returns: Return value type and description
    - Raises: Exceptions that may be raised (if applicable)
    - Note: Additional usage notes (if applicable)
*   **Docstring Style:** Use Google-style docstrings optimized for `python -m pydoc`
*   **Example:**
    ```python
    def create_bookmark(url, title, description, folder_id, tag_ids, favicon=None):
        """Create a new bookmark.

        Args:
            url (str): Bookmark URL
            title (str): Bookmark title
            description (str): Bookmark description
            folder_id (str or None): UUID of parent folder, or None for no folder
            tag_ids (list): List of tag UUIDs to associate with bookmark
            favicon (str, optional): Path to cached favicon image

        Returns:
            str: UUID of the newly created bookmark
        """
    ```
*   **Viewing Documentation:** Use `python -m pydoc <module_name>` to view formatted documentation

## Terraform Coding Style

Follow these coding style rules when writing Terraform code:

*   **Format:** Code must pass `terraform fmt -recursive -check -diff gcp`
*   **Linter:** Code must pass `tflint --chdir gcp`
*   **Security:** Code must pass `tfsec gcp`
*   **Spaces:** Indent with spaces

## Bash and Shell Script Coding Style

Follow these coding style rules when writing Terraform code:

*   **Linter:** Code must pass `shellcheck tools/*.sh && shellcheck gcp/*.sh`
*   **Tabs:** Indent with tabs

## Application Features

### Core Functionality
1. **Bookmarks**: Create, edit, delete bookmarks with URL, title, description, favicon
2. **Folders**: Hierarchical folder structure with unlimited nesting
3. **Tags**: Multiple tags per bookmark
4. **Pinning**: Pin bookmarks to keep them at top of lists
5. **Search**: Full-text search across all fields with live results
6. **Sorting**: Sort by title, URL, or creation date (asc/desc)

### User Experience
1. **Keyboard Shortcuts**:
   - `⌘+/` or `Ctrl+/`: Focus search (OS-aware)
   - `g` then `n`: Add bookmark (Gmail-style)
   - `↑`/`↓`: Navigate search results
   - `Enter`: Open result or submit
   - `Escape`: Close dropdown

2. **Live Search**:
   - Real-time results as you type
   - Keyboard navigation
   - Shows folder context
   - "View all results" link

3. **Automatic Features**:
   - Favicon fetching and caching
   - Title extraction from URLs
   - Folder context preservation

### Design Principles
- **Form-based**: Use HTML forms instead of JavaScript AJAX where possible
- **Progressive Enhancement**: Works without JavaScript, enhanced with it
- **URL-based Navigation**: All views accessible via URL parameters
- **Minimal JavaScript**: Only for live search and keyboard shortcuts
- **Bootstrap Native**: Use Bootstrap components without custom CSS
- **Mobile-First**: Responsive design optimized for all screen sizes

## Testing

Follow these guidelines when working with tests:

*   **Test Framework:** Use pytest for all test cases
*   **Test Location:** Write test cases in the `tests/unit/` directory
*   **Running Tests:** Always run tests after making changes using `python -m pytest tests/ -v`
*   **Test Coverage:** When adding new features or modifying existing code, write corresponding test cases
*   **Test Verification:** After writing test cases, run them to ensure they pass
*   **Test Database:** Tests use in-memory SQLite database with check_same_thread=False

Example commands:
```bash
# Activate virtual environment
source .venv/bin/activate

# Run all tests
python -m pytest tests/ -v

# Run specific test file
python -m pytest tests/unit/test_app.py -v

# Run tests with coverage
python -m pytest tests/ -v --cov=app

# Run tests quietly (summary only)
python -m pytest tests/ -q
```

## Code Modification Guidelines

When modifying code:

1. **Preserve Form-Based Approach**: Don't add unnecessary JavaScript
2. **Use Bootstrap Native**: Don't create custom CSS for layouts
3. **Maintain Keyboard Shortcuts**: Keep existing shortcuts working
4. **UUID Consistency**: Use UUIDs for all database primary keys
5. **Service Layer**: Put business logic in services, not routes
6. **Test Coverage**: Add tests for new features
7. **Mobile Responsive**: Test on mobile viewports
