#!/bin/sh
# Wrap the verified release archive in native packages; never publish.
set -eu
if [ "$#" -ne 1 ]; then
    echo 'Usage: sh scripts/package-native.sh vMAJOR.MINOR[.PATCH]' >&2
    exit 2
fi
tag=$1
if ! printf '%s\n' "$tag" | LC_ALL=C grep -Eq '^v[0-9]+\.[0-9]+(\.[0-9]+)?$'; then
    echo 'native: invalid release tag' >&2; exit 2
fi
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"
command -v nfpm >/dev/null || { echo 'native: nFPM 2.47.0 is required' >&2; exit 1; }
archive="dist/pokecrt_${tag}_linux_amd64.tar.gz"
[ -s "$archive" ] || { echo 'native: build the release archive with scripts/package.sh first' >&2; exit 1; }
version=${tag#v}
case "$version" in *.*.*) ;; *) version="$version.0" ;; esac
stage=$(mktemp -d "$project_root/dist/.native.XXXXXX")
trap 'rm -rf "$stage"' EXIT HUP INT TERM
tar -xzf "$archive" -C "$stage"
[ "$("$stage/pokecrt" --version | head -n 1)" = "pokecrt $tag" ] || { echo 'native: archive version mismatch' >&2; exit 1; }
mkdir "$stage/documentation"
for file in README.md LICENSE LICENSING.md THIRD_PARTY_NOTICES.md docs tools; do
    mv "$stage/$file" "$stage/documentation/"
done
export POKECRT_PACKAGE_ROOT="$stage" POKECRT_PACKAGE_VERSION="$version"
python3 - "$stage/nfpm.json" <<'PYTHON'
import json, os, sys
from pathlib import Path
config = json.loads(Path('packaging/nfpm.json').read_text())
def expand(value):
    if isinstance(value, dict): return {key: expand(item) for key, item in value.items()}
    if isinstance(value, list): return [expand(item) for item in value]
    if isinstance(value, str):
        for key in ('POKECRT_PACKAGE_ROOT', 'POKECRT_PACKAGE_VERSION'):
            value = value.replace('${' + key + '}', os.environ[key])
    return value
Path(sys.argv[1]).write_text(json.dumps(expand(config)))
PYTHON
for format in deb rpm archlinux; do
    case "$format" in
      deb) filename="pokecrt_${version}-1_amd64.deb" ;;
      rpm) filename="pokecrt-${version}-1.x86_64.rpm" ;;
      archlinux) filename="pokecrt-${version}-1-x86_64.pkg.tar.zst" ;;
    esac
    [ ! -e "dist/$filename" ] || { echo "native: output already exists: dist/$filename" >&2; exit 1; }
    nfpm package --config "$stage/nfpm.json" --packager "$format" --target "$stage/$filename"
done
for file in "$stage"/*.deb "$stage"/*.rpm "$stage"/*.pkg.tar.zst; do mv "$file" dist/; done
(cd dist && sha256sum pokecrt_*_linux_amd64.tar.gz pokecrt_*.deb pokecrt-*.rpm pokecrt-*.pkg.tar.zst > SHA256SUMS)
printf '%s\n' 'Native packages prepared. Verify checksums and installation before publication.'
