#!/bin/sh
set -eu
project_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output="$project_root/website/dist"
mkdir -p "$output/assets"
for file in index.html style.css app.js favicon.svg; do
    cp "$project_root/website/$file" "$output/$file"
done
for file in logo-monitor.png logo-wordmark.png home.png pokedex.png encounter.png trainer.png achievements.png cli-artwork.png cli-forms.png demo.mp4; do
    cp "$project_root/docs/media/$file" "$output/assets/$file"
done
: > "$output/.nojekyll"
printf '%s\n' "Website built in $output"
