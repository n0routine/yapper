#!/bin/sh

set -e

for bin in iconv dos2unix; do
    command -v "$bin" >/dev/null 2>/dev/null || { echo "ERROR: no $bin"; exit 1; };
done;

tmp="$(mktemp)"
for file in "$@"; do
    echo "converting $file";
    cat "$file" | iconv -t "utf-8" | tr -cd '[:print:]\n' | dos2unix -r > "$tmp";
    cat "$tmp" > "$file";
done;
rm -rf "$tmp";
