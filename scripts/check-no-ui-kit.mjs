#!/usr/bin/env node
// Fails when a third-party UI kit creeps back into the frontend or docs workspaces.
//
//   node scripts/check-no-ui-kit.mjs <workspaceDir>...
//
// The UI is built from the internal primitives in <workspace>/src/ui. This checks the manifest
// and every source file for the banned packages.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const BANNED = ['primevue', '@primevue/', 'primeicons'];
const found = [];

/** Lists every file below `root`, skipping dependencies and build output. */
function walk(root) {
  return readdirSync(root).flatMap((name) => {
    if (['node_modules', 'dist', 'coverage', 'reports'].includes(name)) return [];
    const p = join(root, name);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });
}

for (const ws of process.argv.slice(2)) {
  const manifest = JSON.parse(readFileSync(join(ws, 'package.json'), 'utf8'));
  const deps = Object.keys({ ...manifest.dependencies, ...manifest.devDependencies });
  for (const d of deps)
    if (BANNED.some((b) => d.startsWith(b))) found.push(`${ws}/package.json: ${d}`);

  for (const file of walk(join(ws, 'src'))) {
    if (!/\.(vue|ts|css)$/.test(file)) continue;
    const text = readFileSync(file, 'utf8');
    for (const b of BANNED) if (text.includes(b)) found.push(`${file}: ${b}`);
  }
}

if (found.length > 0) {
  console.error(`Third-party UI kit found:\n${found.join('\n')}`);
  process.exit(1);
}
console.log('no third-party UI kit in', process.argv.slice(2).join(', '));
