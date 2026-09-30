#!/bin/sh
# Build a local Linux amd64 release candidate. This script never publishes.
set -eu

if [ "$#" -ne 1 ]; then
    echo 'Usage: sh scripts/package.sh vMAJOR.MINOR[.PATCH]' >&2
    exit 2
fi
release_tag=$1
if ! printf '%s\n' "$release_tag" | LC_ALL=C grep -Eq '^v[0-9]+\.[0-9]+(\.[0-9]+)?$'; then
    echo 'package: invalid release tag' >&2
    exit 2
fi
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$project_root"

for required_file in LICENSE LICENSING.md README.md THIRD_PARTY_NOTICES.md tools/dataset/coverage.md; do
    if [ ! -s "$required_file" ]; then
        echo "package: missing required file: $required_file" >&2
        exit 1
    fi
done

# Generation is a separate developer step. This check never fetches or writes.
go run ./tools/dataset \
    --sources tools/dataset/sources.json \
    --cache .cache/dataset \
    --mappings tools/dataset/mappings.json \
    --out . --check
go test ./...
go vet ./...

mkdir -p dist
staging_dir=$(mktemp -d "$project_root/dist/.package.XXXXXX")
trap 'rm -rf "$staging_dir"' EXIT HUP INT TERM
archive_name="pokecrt_${release_tag}_linux_amd64.tar.gz"
if [ -e "dist/$archive_name" ]; then
    echo "package: output already exists: dist/$archive_name" >&2
    exit 1
fi
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$release_tag" \
    -o "$staging_dir/pokecrt" ./cmd/pokecrt
chmod 0755 "$staging_dir/pokecrt"
cp README.md LICENSE LICENSING.md THIRD_PARTY_NOTICES.md "$staging_dir/"
cp tools/dataset/coverage.md "$staging_dir/COVERAGE.md"
tar -czf "dist/$archive_name" -C "$staging_dir" \
    pokecrt README.md LICENSE LICENSING.md THIRD_PARTY_NOTICES.md COVERAGE.md
# Include all local release archives so earlier checksum entries remain present.
(cd dist && sha256sum pokecrt_*_linux_amd64.tar.gz > SHA256SUMS)
printf 'Created dist/%s and dist/SHA256SUMS\n' "$archive_name"
printf 'Local candidate only: complete licensing and release smoke checks before publishing.\n'
