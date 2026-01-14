
def test_add_folder_form_preselects_parent(client, auth_headers, app):
    """Test that adding a folder with parent param pre-selects the parent."""
    # Create a parent folder
    client.post('/folder/save', data={'name': 'Parent Folder'}, headers=auth_headers)
    
    with app.app_context():
        from app.services import folder_service
        folders = folder_service.get_all_folders()
        parent_id = folders[0]['id']

    # Get add form with parent param
    response = client.get(f'/folder/add?parent={parent_id}', headers=auth_headers)
    assert response.status_code == 200
    
    # Check if the option is selected
    html = response.data.decode('utf-8')
    # The template renders: <option value="{{ f.id }}" ... selected>
    # We look for the value and the selected attribute.
    # Note: Jinja might put spaces or not. My edit:
    # <option value="{{ f.id }}" {% if ... %}selected{% endif %}>
    # If selected: <option value="ID" selected>
    # There is a space before {% if %}.
    
    expected_snippet = f'value="{parent_id}" selected'
    assert expected_snippet in html
