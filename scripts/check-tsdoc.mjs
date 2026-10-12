#!/usr/bin/env node
// Fails when an exported declaration has no doc comment.
//
//   node scripts/check-tsdoc.mjs [--list] <srcDir>...
//
// Covers exported functions, classes, interfaces, type aliases, enums and
// top-level const/let declarations in .ts files. Tests, specs, declaration
// files and generated bindings (wailsjs) are skipped. Every Vue single-file
// component must open with an HTML comment describing it (and its props/emits).
import { createRequire } from 'node:module';
import { readdirSync, readFileSync, statSync } from 'node:fs';
import { join } from 'node:path';

const require = createRequire(join(process.cwd(), 'package.json'));
const ts = require('typescript');

const args = process.argv.slice(2);
const list = args.includes('--list');
const roots = args.filter((a) => !a.startsWith('--'));

const skip = (p) =>
  /\.(test|spec)\.ts$/.test(p) ||
  p.endsWith('.d.ts') ||
  p.includes('/wailsjs/') ||
  p.includes('/node_modules/') ||
  p.includes('/test/');

function* walk(dir) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name);
    if (statSync(p).isDirectory()) {
      if (name !== 'node_modules') yield* walk(p);
    } else if ((p.endsWith('.ts') || p.endsWith('.vue')) && !skip(p)) yield p;
  }
}

const missing = [];
for (const root of roots) {
  for (const file of walk(root)) {
    if (file.endsWith('.vue')) {
      if (!readFileSync(file, 'utf8').trimStart().startsWith('<!--')) {
        missing.push(`${file}:1 (component without a leading <!-- description -->)`);
      }
      continue;
    }
    const sf = ts.createSourceFile(file, readFileSync(file, 'utf8'), ts.ScriptTarget.Latest, true);
    for (const node of sf.statements) {
      const exported = node.modifiers?.some((m) => m.kind === ts.SyntaxKind.ExportKeyword);
      if (!exported) continue;
      const documented =
        ts.getJSDocCommentsAndTags(node).length > 0 || hasLeadingLineComment(sf, node);
      if (documented) continue;
      const name =
        node.name?.text ??
        node.declarationList?.declarations.map((d) => d.name.getText(sf)).join(', ') ??
        '(anonymous)';
      const { line } = sf.getLineAndCharacterOfPosition(node.getStart(sf));
      missing.push(`${file}:${line + 1} ${name}`);
    }
  }
}

// A `//` comment directly above the declaration counts as documentation too.
function hasLeadingLineComment(sf, node) {
  const ranges = ts.getLeadingCommentRanges(sf.text, node.getFullStart()) ?? [];
  return ranges.length > 0;
}

if (list || missing.length) console.log(missing.join('\n'));
console.log(`${missing.length} exported declaration(s) without a doc comment`);
process.exit(missing.length ? 1 : 0);
