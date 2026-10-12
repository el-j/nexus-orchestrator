#!/usr/bin/env node
// Bundle budget gate.
//
//   node scripts/check-bundle-budget.mjs <buildDir> <maxJsKB> <maxCssKB>
//   node scripts/check-bundle-budget.mjs <buildDir>            (report only)
//
// Sums the gzip size of every .js and .css file under <buildDir> and fails when a budget
// (in KiB, gzipped) is exceeded. Budgets are set ~15% above the measured size after the
// the UI-library removal; raise one only deliberately and say why in the commit.
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join, extname } from 'node:path';
import { gzipSync } from 'node:zlib';

const [dir, maxJs, maxCss] = process.argv.slice(2);
if (!dir) {
  console.error('usage: check-bundle-budget.mjs <buildDir> [maxJsKB maxCssKB]');
  process.exit(2);
}

/** Lists every file below `root`. */
function walk(root) {
  return readdirSync(root).flatMap((name) => {
    const p = join(root, name);
    return statSync(p).isDirectory() ? walk(p) : [p];
  });
}

const kib = { '.js': 0, '.css': 0 };
for (const file of walk(dir)) {
  const ext = extname(file);
  if (ext in kib) kib[ext] += gzipSync(readFileSync(file)).length / 1024;
}

console.log(`${dir}: JS ${kib['.js'].toFixed(1)} KiB gz, CSS ${kib['.css'].toFixed(1)} KiB gz`);
if (maxJs === undefined) process.exit(0);

let failed = false;
for (const [ext, max] of [
  ['.js', Number(maxJs)],
  ['.css', Number(maxCss)],
]) {
  if (kib[ext] > max) {
    console.error(`FAIL: ${ext} ${kib[ext].toFixed(1)} KiB gz exceeds the ${max} KiB budget`);
    failed = true;
  }
}
process.exit(failed ? 1 : 0);
