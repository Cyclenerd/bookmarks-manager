#!/usr/bin/env bash

# Download Bootstrap CSS to static folder

BOOTSTRAP_VERSION="5.3.8"
# https://www.jsdelivr.com/package/npm/bootstrap
BOOTSTRAP_CSS_URL="https://cdn.jsdelivr.net/npm/bootstrap@${BOOTSTRAP_VERSION}/dist/css/bootstrap.min.css"

BOOTSTRAP_ICONS_VERSION="1.13.1"
# https://www.jsdelivr.com/package/npm/bootstrap-icons
BOOTSTRAP_ICONS_URL="https://cdn.jsdelivr.net/npm/bootstrap-icons@${BOOTSTRAP_ICONS_VERSION}/font/bootstrap-icons.min.css"
BOOTSTRAP_ICONS_WOFF_URL="https://cdn.jsdelivr.net/npm/bootstrap-icons@${BOOTSTRAP_ICONS_VERSION}/font/fonts/bootstrap-icons.woff"
BOOTSTRAP_ICONS_WOFF2_URL="https://cdn.jsdelivr.net/npm/bootstrap-icons@${BOOTSTRAP_ICONS_VERSION}/font/fonts/bootstrap-icons.woff2"

STATIC_DIR="../app/static/css"

# Check that script is executed from tools folder
if [ ! -d "$STATIC_DIR" ]; then
	echo "✗ Error: Script must be executed from the tools folder"
	echo "  $STATIC_DIR directory not found"
	exit 1
fi

echo "Downloading Bootstrap ${BOOTSTRAP_VERSION} CSS..."

# Download Bootstrap CSS

if curl -L -o "${STATIC_DIR}/bootstrap.min.css" "$BOOTSTRAP_CSS_URL"; then
	echo "✓ Bootstrap CSS downloaded successfully to ${STATIC_DIR}/bootstrap.min.css"
else
	echo "✗ Failed to download Bootstrap CSS"
	exit 1
fi

echo "Downloading Bootstrap Icons ${BOOTSTRAP_ICONS_VERSION} CSS..."

# Download Bootstrap Icons CSS

if curl -L -o "${STATIC_DIR}/bootstrap-icons.min.css" "$BOOTSTRAP_ICONS_URL"; then
	echo "✓ Bootstrap Icons CSS downloaded successfully to ${STATIC_DIR}/bootstrap-icons.min.css"
else
	echo "✗ Failed to download Bootstrap Icons CSS"
	exit 1
fi

# Download Bootstrap Icons Fonts

echo "Downloading Bootstrap Icons ${BOOTSTRAP_ICONS_VERSION} Fonts..."

FONTS_DIR="${STATIC_DIR}/fonts"
mkdir -p "$FONTS_DIR"

if curl -L -o "${FONTS_DIR}/bootstrap-icons.woff" "$BOOTSTRAP_ICONS_WOFF_URL"; then
	echo "✓ Bootstrap Icons WOFF downloaded successfully to ${FONTS_DIR}/bootstrap-icons.woff"
else
	echo "✗ Failed to download Bootstrap Icons WOFF"
	exit 1
fi

if curl -L -o "${FONTS_DIR}/bootstrap-icons.woff2" "$BOOTSTRAP_ICONS_WOFF2_URL"; then
	echo "✓ Bootstrap Icons WOFF2 downloaded successfully to ${FONTS_DIR}/bootstrap-icons.woff2"
else
	echo "✗ Failed to download Bootstrap Icons WOFF2"
	exit 1
fi
