#!/usr/bin/env bash

set -eu

# Move to repository root
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR/.."

# Build bakashier binary
sh scripts/build.sh

# Prepare distribution directory
DIST_DIR="dist"
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Copy distributable files
cp -f "./bakashier" "$DIST_DIR/bakashier"
cp -f "./LICENSE" "$DIST_DIR/LICENSE"
cp -f "./THIRD_PARTY_LICENSES.md" "$DIST_DIR/THIRD_PARTY_LICENSES.md"
cp -f "./README.md" "$DIST_DIR/README.md"
cp -f "./README.ja.md" "$DIST_DIR/README.ja.md"
cp -r "./third_party_licenses" "$DIST_DIR/third_party_licenses"
chmod +x "$DIST_DIR/bakashier"

printf 'Created "%s"\n' "$DIST_DIR"
