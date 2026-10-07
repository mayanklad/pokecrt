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

release_notes="docs/release-${release_tag}.md"
for required_file in "$release_notes" docs/guide.md docs/media/home.svg docs/media/logo.png docs/media/logo-wordmark.png docs/media/logo-monitor.png docs/media/home.png docs/media/pokedex.png docs/media/cli-artwork.png docs/media/cli-forms.png docs/media/encounter.png docs/media/trainer.png docs/media/achievements.png docs/media/demo.gif docs/media/demo.mp4 docs/progress.md docs/benchmarks.md docs/release-v0.1.md docs/release-v0.2.md docs/release-v0.3.md docs/release-v0.4.md LICENSE LICENSING.md README.md THIRD_PARTY_NOTICES.md tools/dataset/coverage.md docs/release-policy.md; do
    if [ ! -s "$required_file" ]; then
        echo "package: missing required file: $required_file" >&2
        exit 1
    fi
done

# Materialize ignored sprites from verified cached inputs; reject metadata drift.
# Populate the cache explicitly with the developer --fetch command first.
go run ./tools/dataset \
    --sources tools/dataset/sources.json \
    --cache .cache/dataset \
    --mappings tools/dataset/mappings.json \
    --out . --prepare-assets --check
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
mkdir -p "$staging_dir/docs/media" "$staging_dir/tools/dataset"
cp tools/dataset/coverage.md "$staging_dir/tools/dataset/coverage.md"
cp docs/guide.md docs/progress.md docs/benchmarks.md docs/release-policy.md \
    docs/release-v0.1.md docs/release-v0.2.md docs/release-v0.3.md docs/release-v0.4.md \
    "$release_notes" "$staging_dir/docs/"
cp docs/media/home.svg docs/media/logo.png docs/media/logo-wordmark.png docs/media/logo-monitor.png docs/media/home.png docs/media/pokedex.png \
    docs/media/cli-artwork.png docs/media/cli-forms.png docs/media/encounter.png \
    docs/media/trainer.png docs/media/achievements.png docs/media/demo.gif docs/media/demo.mp4 \
    "$staging_dir/docs/media/"
tar -czf "dist/$archive_name" -C "$staging_dir" \
    pokecrt README.md LICENSE LICENSING.md THIRD_PARTY_NOTICES.md docs tools/dataset/coverage.md
# Include all local release archives so earlier checksum entries remain present.
(cd dist && sha256sum pokecrt_*_linux_amd64.tar.gz > SHA256SUMS)
printf 'Created dist/%s and dist/SHA256SUMS\n' "$archive_name"
printf 'Release candidate prepared: verify final tag, checksums, and extracted-binary smoke checks before publishing.\n'
