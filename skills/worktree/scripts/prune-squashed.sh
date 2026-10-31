#!/usr/bin/env bash
set -euo pipefail

# Prune local branches already squash-merged into target branch (default: main)
target="${1:-main}"

git checkout -q "$target"
for branch in $(git for-each-ref --format='%(refname:short)' refs/heads/); do
  [ "$branch" = "$target" ] && continue
  base=$(git merge-base "$target" "$branch")
  tree=$(git rev-parse "$branch^{tree}")
  commit=$(git commit-tree "$tree" -p "$base" -m "_")
  if [ "$(git cherry "$target" "$commit" | cut -c1)" = "-" ]; then
    git branch -D "$branch"
  fi
done
