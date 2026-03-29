#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

BRANCH="${BRANCH:-main}"
REMOTE="${REMOTE:-origin}"
MESSAGE="${1:-chore: auto push $(date '+%Y-%m-%d %H:%M:%S')}"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: current directory is not a git repository" >&2
  exit 1
fi

if ! git remote get-url "$REMOTE" >/dev/null 2>&1; then
  echo "error: git remote '$REMOTE' is not configured" >&2
  exit 1
fi

git add -A

if git diff --cached --quiet; then
  echo "nothing to commit"
  exit 0
fi

git commit -m "$MESSAGE"
git push -u "$REMOTE" "$BRANCH"
