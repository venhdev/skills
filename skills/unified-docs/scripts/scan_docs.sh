#!/usr/bin/env bash
set -euo pipefail

root="${1:-.}"

find "$root" \
  \( -path '*/.git' -o -path '*/node_modules' -o -path '*/dist' -o -path '*/build' \) -prune -o \
  -type f \( -name '*.md' -o -name '*.mdx' \) \
  -not -name 'README.md' -not -name 'CHANGELOG.md' -print | sort
