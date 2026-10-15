#!/usr/bin/env bash
# next-version.sh — the version number a pre-release build of the current commit gets.
#
# Stable versions are made by release-please (see docs/releasing.md): a clean X.Y.Z, never with a
# trailing "-N". Builds of `develop` are betas of the NEXT minor version:
#
#   last stable (the release-please manifest on main) 0.10.0  →  0.11.0-beta.<N>
#
# N counts the commits since the last stable tag and is raised above every beta that exists, so it
# only grows. Needs the full history and the tags (fetch-depth: 0).
#
# Usage: scripts/next-version.sh        prints e.g. 0.11.0-beta.42
# Environment: NEXUS_MANIFEST_REF  git ref that holds .release-please-manifest.json (default origin/main)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
ref="${NEXUS_MANIFEST_REF:-origin/main}"

manifest="$(git show "${ref}:.release-please-manifest.json" 2>/dev/null || cat .release-please-manifest.json)"
last="$(printf '%s' "$manifest" | sed -n 's/.*"\.": *"\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\)".*/\1/p' | head -1)"
if [[ -z "$last" ]]; then
  echo "next-version: no version for \".\" in .release-please-manifest.json" >&2
  exit 1
fi

IFS=. read -r major minor _patch <<<"$last"
base="${major}.$((minor + 1)).0"

if git rev-parse -q --verify "refs/tags/v${last}" >/dev/null; then
  n="$(git rev-list --count "v${last}..HEAD")"
else
  n="$(git rev-list --count HEAD)"
fi
[[ "$n" -ge 1 ]] || n=1
# never go below a beta that already exists (the numbers only grow)
top="$(git tag --list "v${base}-beta.*" | sed 's/.*beta\.//' | grep -E '^[0-9]+$' | sort -n | tail -1 || true)"
if [[ -n "$top" && "$n" -le "$top" ]]; then n=$((top + 1)); fi
while git rev-parse -q --verify "refs/tags/v${base}-beta.${n}" >/dev/null; do
  n=$((n + 1))
done
echo "${base}-beta.${n}"
