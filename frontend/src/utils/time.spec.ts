import { describe, it, expect, vi, afterEach } from 'vitest';
import { relativeTime, timeAgo, formatDate } from './time';

afterEach(() => vi.useRealTimers());

describe('relativeTime', () => {
  const now = new Date('2026-06-15T12:00:00Z');
  const ago = (ms: number) => new Date(now.getTime() - ms).toISOString();

  it.each([
    [undefined, '—'],
    ['', '—'],
    [30_000, 'just now'],
    [5 * 60_000, '5 min ago'],
    [3 * 3_600_000, '3 hr ago'],
  ])('formats %s as %s', (offset, expected) => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const input = typeof offset === 'number' ? ago(offset) : (offset as string | undefined);
    expect(relativeTime(input)).toBe(expected);
  });

  it('falls back to a calendar date after a day', () => {
    vi.useFakeTimers();
    vi.setSystemTime(now);
    const iso = ago(3 * 86_400_000);
    expect(relativeTime(iso)).toBe(new Date(iso).toLocaleDateString());
  });

  it('keeps the legacy aliases', () => {
    expect(timeAgo).toBe(relativeTime);
    expect(formatDate).toBe(relativeTime);
  });
});
