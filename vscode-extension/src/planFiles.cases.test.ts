import {
  mkdtempSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'fs';
import { tmpdir } from 'os';
import * as path from 'path';
import { describe, expect, it } from 'vitest';
import { scanPlanFiles, type PlanFilesConfig, type PlanFs } from './planFiles';

interface Case {
  name: string;
  config?: PlanFilesConfig;
  files: Record<string, string>;
  expect: {
    plans: Array<{ id: string; title: string; status: string; tasks: string[] }>;
    orphanTasks: string[];
    folders: string[];
  };
}

const realFs: PlanFs = {
  readdirSync: (dir) =>
    readdirSync(dir, { withFileTypes: true }).map((e) => ({
      name: e.name,
      isDirectory: e.isDirectory(),
    })),
  readFileSync: (p) => readFileSync(p, 'utf8'),
  statSync: (p) => statSync(p),
};

// The same cases run against the daemon (internal/core/services/planfiles_cases_test.go): both
// must find the same plans, tasks and links in the same files.
const doc = JSON.parse(
  readFileSync(path.resolve(__dirname, '../../testdata/planfiles/cases.json'), 'utf8'),
) as { cases: Case[] };

describe('shared plan-file cases', () => {
  it.each(doc.cases.map((c) => [c.name, c] as const))('%s', (_name, c) => {
    const dir = mkdtempSync(path.join(tmpdir(), 'nexus-plans-'));
    try {
      for (const [rel, text] of Object.entries(c.files)) {
        mkdirSync(path.dirname(path.join(dir, rel)), { recursive: true });
        writeFileSync(path.join(dir, rel), text);
      }
      const got = scanPlanFiles(dir, realFs, c.config);
      expect(
        got.plans.map((p) => ({
          id: p.id,
          title: p.title,
          status: p.status,
          tasks: p.tasks.map((t) => t.id),
        })),
      ).toEqual(c.expect.plans);
      expect(got.orphans.map((t) => t.id)).toEqual(c.expect.orphanTasks);
      expect(got.folders).toEqual(c.expect.folders);
    } finally {
      rmSync(dir, { recursive: true, force: true });
    }
  });
});
