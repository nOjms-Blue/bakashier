#!/usr/bin/env bash

set -eu

# Move to repository root
SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR/.."

# Build bakashier binary
if ! sh scripts/build.sh; then
  exit 1
fi

# Install binary
INSTALL_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/bakashier"
mkdir -p "$INSTALL_DIR"
cp -f "./bakashier" "$INSTALL_DIR/bakashier"
cp -f "./LICENSE" "$INSTALL_DIR/LICENSE"
cp -f "./NOTICE" "$INSTALL_DIR/NOTICE"
cp -f "./README.md" "$INSTALL_DIR/README.md"
cp -f "./README.ja.md" "$INSTALL_DIR/README.ja.md"
chmod +x "$INSTALL_DIR/bakashier"

# Add install dir to PATH in shell rc files if needed
case ":$PATH:" in
  *":$INSTALL_DIR:"*)
    ;;
  *)
    LINE="export PATH=\"$INSTALL_DIR:\$PATH\""
    for rc in "$HOME/.profile" "$HOME/.bashrc" "$HOME/.zshrc"; do
      touch "$rc"
      grep -F "$LINE" "$rc" >/dev/null 2>&1 || printf '\n%s\n' "$LINE" >>"$rc"
    done
    ;;
esac
