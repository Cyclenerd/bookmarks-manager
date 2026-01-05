"""Main routes module for the bookmarks application.

This module defines all HTTP routes and view functions for the Flask application,
including bookmark management, folder operations, tag handling, search functionality,
and Firefox import/export features.
"""

from flask import Blueprint, render_template, send_from_directory, request, redirect, url_for, jsonify, Response
from app.utils.auth import requires_auth
from app.services import bookmark_service, folder_service, tag_service, favicon_service, metadata_service, firefox_service
import json

bp = Blueprint('main', __name__)


@bp.route('/')
@requires_auth
def index():
    """Render the main bookmarks listing page.

    Displays bookmarks filtered by folder or tag with pagination and sorting support.
    If a search query is provided, redirects to the dedicated search page.

    Query Parameters:
        folder (str): Optional folder ID to filter bookmarks (use 'unfiled' for bookmarks without a folder)
        tag (str): Optional tag ID to filter bookmarks
        search (str): Optional search query (redirects to search page)
        sort (str): Sort field (title, url, created_at), default: created_at
        order (str): Sort order (asc, desc), default: desc
        page (int): Page number for pagination, default: 1

    Returns:
        str: Rendered HTML template with bookmarks and navigation data
    """
    folders = folder_service.get_folder_hierarchy()
    tags = tag_service.get_all_tags()
    folder_id = request.args.get('folder')
    tag_id = request.args.get('tag')
    search = request.args.get('search', '')
    sort_by = request.args.get('sort', 'created_at')
    sort_order = request.args.get('order', 'desc')
    page = request.args.get('page', 1, type=int)

    # Check if we're viewing unfiled bookmarks
    is_unfiled = folder_id == 'unfiled'

    # If search query exists, redirect to search page
    if search:
        return redirect(url_for('main.search_page', q=search))

    result = bookmark_service.get_all_bookmarks(
        folder_id=folder_id,
        tag_id=tag_id,
        search=None,
        sort_by=sort_by,
        sort_order=sort_order,
        include_subfolders=False,
        page=page,
        per_page=25
    )

    current_folder = None
    current_tag = None
    if folder_id and not is_unfiled:
        current_folder = folder_service.get_folder(folder_id)
    if tag_id:
        current_tag = tag_service.get_tag(tag_id)

    return render_template('index.html',
                           bookmarks=result['bookmarks'],
                           folders=folders,
                           tags=tags,
                           current_folder=current_folder,
                           current_tag=current_tag,
                           search=search,
                           sort_by=sort_by,
                           sort_order=sort_order,
                           page=result['page'],
                           total_pages=result['total_pages'],
                           total=result['total'],
                           is_unfiled=is_unfiled)


@bp.route('/robots.txt')
def robots():
    """Serve robots.txt file."""
    return send_from_directory('static', 'robots.txt', mimetype='text/plain')


@bp.route('/favicon.ico')
def favicon():
    """Serve favicon.ico file."""
    return send_from_directory('static/img', 'favicon.ico', mimetype='image/vnd.microsoft.icon')


@bp.route('/search')
@requires_auth
def search_page():
    """Render the search results page.

    Displays full-text search results across all bookmarks with pagination and sorting.

    Query Parameters:
        q (str): Search query string
        sort (str): Sort field (title, url, created_at), default: created_at
        order (str): Sort order (asc, desc), default: desc
        page (int): Page number for pagination, default: 1

    Returns:
        str: Rendered HTML template with search results
    """
    query = request.args.get('q', '')
    sort_by = request.args.get('sort', 'created_at')
    sort_order = request.args.get('order', 'desc')
    page = request.args.get('page', 1, type=int)

    result = {'bookmarks': [], 'total': 0, 'page': 1, 'total_pages': 0}
    if query:
        result = bookmark_service.get_all_bookmarks(
            search=query,
            sort_by=sort_by,
            sort_order=sort_order,
            page=page,
            per_page=25
        )

    folders = folder_service.get_folder_hierarchy()
    tags = tag_service.get_all_tags()

    return render_template('search_results.html',
                           bookmarks=result['bookmarks'],
                           folders=folders,
                           tags=tags,
                           query=query,
                           sort_by=sort_by,
                           sort_order=sort_order,
                           page=result['page'],
                           total_pages=result['total_pages'],
                           total=result['total'])


@bp.route('/api/fetch-metadata', methods=['POST'])
@requires_auth
def fetch_metadata():
    """API endpoint to fetch page metadata from a URL.

    Extracts title and other metadata from the provided URL for bookmark creation.

    Request JSON:
        url (str): The URL to fetch metadata from

    Returns:
        JSON: Metadata object with title and success status, or error message
        HTTP 400: If URL is missing from request
    """
    if not request.is_json:
        return jsonify({'success': False, 'error': 'Content-Type must be application/json'}), 400

    url = request.json.get('url')
    if not url:
        return jsonify({'success': False, 'error': 'URL is required'}), 400

    metadata = metadata_service.fetch_page_metadata(url)
    return jsonify(metadata)


@bp.route('/api/search', methods=['GET'])
@requires_auth
def search_api():
    """API endpoint for live bookmark search.

    Provides real-time search results for the autocomplete/live search feature.
    Returns a maximum of 10 results.

    Query Parameters:
        q (str): Search query string (minimum 2 characters)

    Returns:
        JSON: Array of bookmark objects with id, title, url, folder_name, and favicon
    """
    query = request.args.get('q', '')
    if not query or len(query) < 2:
        return jsonify({'bookmarks': []})

    result = bookmark_service.get_all_bookmarks(search=query, sort_by='created_at', sort_order='desc', page=1, per_page=10)
    results = []
    for bookmark in result['bookmarks']:
        results.append({
            'id': bookmark['id'],
            'title': bookmark['title'],
            'url': bookmark['url'],
            'folder_name': bookmark['folder_name'],
            'favicon': bookmark['favicon']
        })

    return jsonify({'bookmarks': results})


@bp.route('/bookmark/add')
@requires_auth
def add_bookmark_form():
    """Display the form for adding a new bookmark.

    Query Parameters:
        folder (str): Optional folder ID to pre-select
        url (str): Optional URL to pre-populate
        title (str): Optional title to pre-populate

    Returns:
        str: Rendered HTML form template
    """
    folder_id = request.args.get('folder')
    url = request.args.get('url')
    title = request.args.get('title')
    folders = folder_service.get_folder_hierarchy()
    tags = tag_service.get_all_tags()
    current_folder = None
    if folder_id:
        current_folder = folder_service.get_folder(folder_id)
    return_url = request.referrer or url_for('main.index')
    return render_template('bookmark_form.html', folders=folders, tags=tags, bookmark=None,
                           current_folder=current_folder, url=url, title=title, return_url=return_url)


@bp.route('/bookmark/<bookmark_id>/edit')
@requires_auth
def edit_bookmark_form(bookmark_id):
    """Display the form for editing an existing bookmark.

    Args:
        bookmark_id (str): UUID of the bookmark to edit

    Returns:
        str: Rendered HTML form template, or redirect to index if bookmark not found
    """
    bookmark = bookmark_service.get_bookmark(bookmark_id)
    if not bookmark:
        return redirect(url_for('main.index'))
    folders = folder_service.get_folder_hierarchy()
    tags = tag_service.get_all_tags()
    return_url = request.referrer or url_for('main.index')
    return render_template('bookmark_form.html', folders=folders, tags=tags, bookmark=bookmark,
                           current_folder=None, return_url=return_url)


@bp.route('/bookmark/save', methods=['POST'])
@requires_auth
def save_bookmark():
    """Save a new or updated bookmark.

    Creates a new bookmark if bookmark_id is not provided, otherwise updates existing.
    Automatically downloads and caches the favicon for the URL.

    Form Data:
        bookmark_id (str): Optional UUID for updating existing bookmark
        url (str): Bookmark URL
        title (str): Bookmark title
        description (str): Optional bookmark description
        folder_id (str): Optional folder UUID
        tag_ids (list): Optional list of tag UUIDs
        return_url (str): Optional URL to redirect after save

    Returns:
        redirect: Redirects to return_url or main index page after save
    """
    bookmark_id = request.form.get('bookmark_id')
    url = request.form.get('url')
    title = request.form.get('title')
    description = request.form.get('description', '')
    folder_id = request.form.get('folder_id')
    tag_ids = request.form.getlist('tag_ids')
    return_url = request.form.get('return_url')

    folder_id = folder_id if folder_id else None
    favicon = favicon_service.download_favicon(url)

    if bookmark_id:
        bookmark_service.update_bookmark(bookmark_id, url, title, description, folder_id, tag_ids, favicon)
    else:
        bookmark_service.create_bookmark(url, title, description, folder_id, tag_ids, favicon)

    return redirect(return_url or url_for('main.index'))


@bp.route('/bookmark/<bookmark_id>/delete', methods=['POST'])
@requires_auth
def delete_bookmark(bookmark_id):
    """Delete a bookmark.

    Args:
        bookmark_id (str): UUID of the bookmark to delete

    Form Data:
        return_url (str): Optional URL to redirect after save

    Returns:
        redirect: Redirects to return_url or referring page
    """
    return_url = request.form.get('return_url')

    bookmark_service.delete_bookmark(bookmark_id)
    return redirect(return_url or request.referrer)


@bp.route('/bookmark/<bookmark_id>/toggle-pin', methods=['POST'])
@requires_auth
def toggle_pin(bookmark_id):
    """Toggle the pinned status of a bookmark.

    Pinned bookmarks appear at the top of bookmark lists.

    Args:
        bookmark_id (str): UUID of the bookmark to toggle

    Returns:
        redirect: Redirects to the referring page or index
    """
    bookmark_service.toggle_pin(bookmark_id)
    return redirect(request.referrer or url_for('main.index'))


@bp.route('/folder/add')
@requires_auth
def add_folder_form():
    """Display the form for adding a new folder.

    Query Parameters:
        parent (str): Optional parent folder ID for nesting

    Returns:
        str: Rendered HTML form template
    """
    parent_id = request.args.get('parent')
    folders = folder_service.get_folder_hierarchy()
    parent_folder = None
    if parent_id:
        parent_folder = folder_service.get_folder(parent_id)
    return_url = request.referrer or url_for('main.index')
    return render_template('folder_form.html', folder=None, folders=folders, parent_folder=parent_folder,
                           return_url=return_url)


@bp.route('/folder/<folder_id>/edit')
@requires_auth
def edit_folder_form(folder_id):
    """Display the form for editing an existing folder.

    Args:
        folder_id (str): UUID of the folder to edit

    Returns:
        str: Rendered HTML form template, or redirect to index if folder not found
    """
    folder = folder_service.get_folder(folder_id)
    if not folder:
        return redirect(url_for('main.index'))
    folders = folder_service.get_folder_hierarchy()
    return_url = request.referrer or url_for('main.index')
    return render_template('folder_form.html', folder=folder, folders=folders, parent_folder=None, return_url=return_url)


@bp.route('/folder/save', methods=['POST'])
@requires_auth
def save_folder():
    """Save a new or updated folder.

    Creates a new folder if folder_id is not provided, otherwise updates existing.
    Validates that folders cannot be moved into their own subfolders.

    Form Data:
        folder_id (str): Optional UUID for updating existing folder
        name (str): Folder name
        parent_id (str): Optional parent folder UUID
        return_url (str): Optional URL to redirect after save

    Returns:
        redirect: Redirects to return_url or main index page after save
    """
    folder_id = request.form.get('folder_id')
    name = request.form.get('name')
    parent_id = request.form.get('parent_id')
    parent_id = parent_id if parent_id else None
    return_url = request.form.get('return_url')

    try:
        if folder_id:
            folder_service.update_folder(folder_id, name, parent_id)
        else:
            folder_service.create_folder(name, parent_id)
    except ValueError:
        if folder_id:
            return redirect(url_for('main.edit_folder_form', folder_id=folder_id))
        else:
            return redirect(return_url or url_for('main.index'))

    return redirect(return_url or url_for('main.index'))


@bp.route('/folder/<folder_id>/delete', methods=['POST'])
@requires_auth
def delete_folder(folder_id):
    """Delete a folder.

    Deletes the folder and sets bookmarks in it to have no folder (NULL).

    Form Data:
        return_url (str): Optional URL to redirect after save

    Args:
        folder_id (str): UUID of the folder to delete

    Returns:
        redirect: Redirects to main index
    """

    folder_service.delete_folder(folder_id)
    return redirect(url_for('main.index'))


@bp.route('/tag/add')
@requires_auth
def add_tag_form():
    """Display the form for adding a new tag.

    Returns:
        str: Rendered HTML form template
    """
    return_url = request.referrer or url_for('main.index')
    return render_template('tag_form.html', tag=None, return_url=return_url)


@bp.route('/tag/<tag_id>/edit')
@requires_auth
def edit_tag_form(tag_id):
    """Display the form for editing an existing tag.

    Args:
        tag_id (str): UUID of the tag to edit

    Returns:
        str: Rendered HTML form template, or redirect to index if tag not found
    """
    tag = tag_service.get_tag(tag_id)
    if not tag:
        return redirect(url_for('main.index'))
    return_url = request.referrer or url_for('main.index')
    return render_template('tag_form.html', tag=tag, return_url=return_url)


@bp.route('/tag/save', methods=['POST'])
@requires_auth
def save_tag():
    """Save a new or updated tag.

    Creates a new tag if tag_id is not provided, otherwise updates existing.

    Form Data:
        tag_id (str): Optional UUID for updating existing tag
        name (str): Tag name
        return_url (str): Optional URL to redirect after save

    Returns:
        redirect: Redirects to return_url or main index page after save
    """
    tag_id = request.form.get('tag_id')
    name = request.form.get('name')
    return_url = request.form.get('return_url')

    if tag_id:
        tag_service.update_tag(tag_id, name)
    else:
        tag_service.create_tag(name)

    return redirect(return_url or url_for('main.index'))


@bp.route('/tag/<tag_id>/delete', methods=['POST'])
@requires_auth
def delete_tag(tag_id):
    """Delete a tag.

    Removes the tag and all bookmark-tag associations.

    Form Data:
        return_url (str): Optional URL to redirect after save

    Args:
        tag_id (str): UUID of the tag to delete

    Returns:
        redirect: Redirects to main index
    """

    tag_service.delete_tag(tag_id)
    return redirect(url_for('main.index'))


@bp.route('/import')
@requires_auth
def import_page():
    """Display the bookmark import page.

    Returns:
        str: Rendered HTML template with import form
    """
    return render_template('import.html')


@bp.route('/import/firefox', methods=['POST'])
@requires_auth
def import_firefox():
    """Import bookmarks from a Firefox JSON file.

    Processes uploaded Firefox bookmark JSON file and imports bookmarks,
    folders, and tags into the application database.

    Form Data:
        file: Firefox JSON bookmark file

    Returns:
        str: Success template with import statistics, or error template on failure
    """
    if 'file' not in request.files:
        return redirect(url_for('main.import_page'))

    file = request.files['file']
    if file.filename == '':
        return redirect(url_for('main.import_page'))

    if not file.filename.lower().endswith('.json'):
        return render_template('import_error.html', error='Only JSON files are allowed')

    try:
        content = file.read().decode('utf-8')
        data = firefox_service.parse_firefox_json(content)
        stats = firefox_service.import_from_firefox_json(data)

        return render_template('import_success.html', stats=stats)
    except UnicodeDecodeError:
        return render_template('import_error.html', error='Invalid file encoding. Must be UTF-8.')
    except Exception as e:
        return render_template('import_error.html', error=str(e))


@bp.route('/export/firefox')
@requires_auth
def export_firefox():
    """Export all bookmarks to Firefox JSON format.

    Generates a Firefox-compatible JSON file containing all bookmarks,
    folders, and tags from the application database.

    Returns:
        Response: JSON file download with bookmarks.json filename
    """
    data = firefox_service.export_to_firefox_json()
    json_str = json.dumps(data, indent=2, ensure_ascii=False)

    response = Response(json_str, mimetype='application/json')
    response.headers['Content-Disposition'] = 'attachment; filename=bookmarks.json'
    return response
