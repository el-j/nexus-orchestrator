#!/usr/bin/env bash
# release.sh — Create and push a semver git tag for a given release track.
#
# Usage:
#   ./scripts/release.sh alpha  0.10.0   → v0.10.0-alpha.1 (auto-increments)
#   ./scripts/release.sh beta   0.10.0   → v0.10.0-beta.1
#   ./scripts/release.sh rc     0.10.0   → v0.10.0-rc.1
#   ./scripts/release.sh stable 0.10.0   → v0.10.0
#
# The script:
#   1. Validates the working tree is clean
#   2. Computes the next N for pre-release tags (auto-increment)
#   3. Creates an annotated git tag
#   4. Syncs package versions
#   5. Prompts before pushing
set -euo pipefail

TRACK="${1:-}"
BASE_VERSION="${2:-}"

if [[ -z "$TRACK" || -z "$BASE_VERSION" ]]; then
  echo "Usage: $0 <alpha|beta|rc|stable> <base-version>" >&2
  echo "  Example: $0 beta 0.10.0" >&2
  exit 1
fi

# Strip leading v if present
BASE_VERSION="${BASE_VERSION#v}"

# Validate track
case "$TRACK" in
  alpha|beta|rc|stable) ;;
  *)
    echo "Error: track must be one of: alpha, beta, rc, stable" >&2
    exit 1
    ;;
esac

# Validate clean working tree
if ! git diff --quiet || ! git diff --cached --quiet; then
  echo "Error: working tree is dirty. Commit or stash changes first." >&2
  git status --short
  exit 1
fi

# Compute tag
if [[ "$TRACK" == "stable" ]]; then
  TAG="v${BASE_VERSION}"
else
  # Find highest existing N for this base+track and increment
  EXISTING=$(git tag --list "v${BASE_VERSION}-${TRACK}.*" | grep -oP '(?<=\.)[0-9]+$' | sort -n | tail -1)
  NEXT=$(( ${EXISTING:-0} + 1 ))
  TAG="v${BASE_VERSION}-${TRACK}.${NEXT}"
fi

echo "→ Creating release tag: ${TAG}"

# Check tag doesn't already exist
if git tag --list | grep -qx "$TAG"; then
  echo "Error: tag ${TAG} already exists." >&2
  exit 1
fi

# Sync package manifests
bash "$(dirname "$0")/version-sync.sh" "${TAG#v}"

# Stage manifest changes
git add frontend/package.json vscode-extension/package.json wails.json 2>/dev/null || true

# Commit if there are staged changes
if ! git diff --cached --quiet; then
  git commit -m "chore: bump manifests to ${TAG#v}"
fi

# Create annotated tag
git tag -a "$TAG" -m "Release ${TAG}"
echo "→ Tag ${TAG} created."

# Prompt to push
echo ""
read -rp "Push tag ${TAG} to origin? [y/N] " CONFIRM
if [[ "${CONFIRM,,}" == "y" ]]; then
  git push origin "$TAG"
  echo "→ Pushed ${TAG} to origin."
else
  echo "→ Not pushed. Run: git push origin ${TAG}"
fi
