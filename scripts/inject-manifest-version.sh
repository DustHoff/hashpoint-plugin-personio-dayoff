#!/bin/sh
# Rewrite the `version = "..."` field in manifest.toml with the value passed
# as $1 and write the result to dist/manifest.toml. Invoked by GoReleaser's
# `before.hooks` so the archived manifest matches the git tag without ever
# mutating the source tree.
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 <version>" >&2
  exit 2
fi

version="$1"
mkdir -p dist
sed "s/^version *= *\"[^\"]*\"/version = \"${version}\"/" manifest.toml > dist/manifest.toml
echo "wrote dist/manifest.toml with version=${version}"
