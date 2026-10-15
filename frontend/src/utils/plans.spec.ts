import { describe, it, expect } from 'vitest';
import type { DiscoveredPlanFile, PlanFileKind } from '../types/domain';
import {
  kindLabel,
  extractParentPlan,
  basename,
  truncate,
  formatDate,
  kindBadgeClass,
  parseStatus,
  statusBadgeClass,
} from './plans';

const plan = (summary?: string) => ({ summary }) as DiscoveredPlanFile;

describe('plan helpers', () => {
  it('labels kinds and falls back to the raw kind', () => {
    expect(kindLabel('claude-task')).toBe('Tasks');
    expect(kindLabel('weird' as PlanFileKind)).toBe('weird');
  });

  it('finds the parent plan id', () => {
    expect(extractParentPlan(plan('x **Plan:** PLAN-12 y'))).toBe('PLAN-12');
    expect(extractParentPlan(plan('nothing'))).toBeNull();
    expect(extractParentPlan(plan(undefined))).toBeNull();
  });

  it('basename and truncate', () => {
    expect(basename('/a/b/c.md')).toBe('c.md');
    expect(basename('')).toBe('');
    expect(truncate('abcdef', 3)).toBe('abc…');
    expect(truncate('abc', 3)).toBe('abc');
  });

  it('formats dates and keeps unparseable input', () => {
    expect(formatDate('2026-06-15T12:00:00Z')).toMatch(/2026/);
    expect(formatDate('not a date')).toBe('Invalid Date');
  });

  it('formatDate returns the input when the runtime throws', () => {
    const original = Date.prototype.toLocaleDateString;
    Date.prototype.toLocaleDateString = () => {
      throw new Error('boom');
    };
    try {
      expect(formatDate('2026-06-15')).toBe('2026-06-15');
    } finally {
      Date.prototype.toLocaleDateString = original;
    }
  });

  it('picks badge classes with a neutral fallback', () => {
    expect(kindBadgeClass('nexus')).toContain('indigo');
    expect(kindBadgeClass('x' as PlanFileKind)).toContain('slate-400');
    expect(statusBadgeClass('done')).toContain('emerald');
    expect(statusBadgeClass('odd')).toContain('slate-400');
  });

  it('parses status from the body before YAML frontmatter', () => {
    expect(parseStatus(plan(undefined))).toBeNull();
    expect(parseStatus(plan('**Status:** TODO'))).toBe('todo');
    expect(parseStatus(plan('---\nstatus: In-Progress\n---'))).toBe('in-progress');
    expect(parseStatus(plan('no status here'))).toBeNull();
  });
});
