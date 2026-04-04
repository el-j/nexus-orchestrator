#!/usr/bin/env bash
# version-sync.sh — stamp the computed semver into all package manifests.
#
# Usage:
#   ./scripts/version-sync.sh 0.10.0
#   ./scripts/version-sync.sh 0.10.0-rc.1
#
# Called by: `make version-sync`, CI publish workflow before packaging.
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <semver>" >&2
  exit 1
fi

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "→ Syncing version ${VERSION} into all package manifests"

# --- frontend/package.json ---
node -e "
  const fs = require('fs');
  const p = '${ROOT}/frontend/package.json';
  const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
  pkg.version = '${VERSION}';
  fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
"
echo "  ✓ frontend/package.json"

# --- vscode-extension/package.json ---
node -e "
  const fs = require('fs');
  const p = '${ROOT}/vscode-extension/package.json';
  const pkg = JSON.parse(fs.readFileSync(p, 'utf8'));
  pkg.version = '${VERSION}';
  fs.writeFileSync(p, JSON.stringify(pkg, null, 2) + '\n');
"
echo "  ✓ vscode-extension/package.json"

# --- wails.json ---
node -e "
  const fs = require('fs');
  const p = '${ROOT}/wails.json';
  const cfg = JSON.parse(fs.readFileSync(p, 'utf8'));
  cfg.info = cfg.info || {};
  cfg.info.productVersion = '${VERSION}';
  fs.writeFileSync(p, JSON.stringify(cfg, null, 2) + '\n');
"
echo "  ✓ wails.json"

echo "→ Done. All manifests now at ${VERSION}"
