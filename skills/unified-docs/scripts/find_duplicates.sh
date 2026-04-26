#!/usr/bin/env bash
set -uo pipefail

if [ "$#" -lt 1 ]; then
  echo "usage: $0 \"Title\" [root]" >&2
  exit 1
fi

title="$1"
root="${2:-.}"
pattern="^title:[[:space:]]*.*${title}|^#[[:space:]]+${title}$"

find "$root" \
  \( -path '*/.git' -o -path '*/node_modules' -o -path '*/dist' -o -path '*/build' \) -prune -o \
  -type f \( -name '*.md' -o -name '*.mdx' \) -print 2>/dev/null |
while IFS= read -r file; do
  if grep -Eis "$pattern" "$file" >/dev/null 2>&1; then
    printf '%s\n' "$file"
  fi
done | sort -u

exit 0
