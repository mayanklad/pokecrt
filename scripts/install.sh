#!/bin/sh
# Install an already-built local executable. No downloads or trainer changes.
set -eu
if [ "$#" -gt 1 ]; then
    echo 'Usage: sh scripts/install.sh [path/to/pokecrt]' >&2
    exit 2
fi
source_binary=${1:-./bin/pokecrt}
install_dir=${POKECRT_INSTALL_DIR:-"$HOME/.local/bin"}
if [ ! -f "$source_binary" ] || [ ! -x "$source_binary" ]; then
    echo "install: executable not found: $source_binary" >&2
    exit 1
fi
# Check that the supplied executable runs before replacing the installed copy.
"$source_binary" --version
mkdir -p "$install_dir"
install -m 0755 "$source_binary" "$install_dir/pokecrt"
printf 'Installed %s/pokecrt\n' "$install_dir"
printf 'Ensure this directory is on PATH.\n'
