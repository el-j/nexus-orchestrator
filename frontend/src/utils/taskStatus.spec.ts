import { describe, it, expect } from 'vitest';
import { HISTORY_FILTERS, HISTORY_STATUSES, STATUS_PRIORITY } from './taskStatus';

describe('taskStatus', () => {
  it('has a filter button for every finished state plus All', () => {
    expect(HISTORY_FILTERS[0].value).toBe('ALL');
    expect(
      HISTORY_FILTERS.slice(1)
        .map((f) => f.value)
        .sort(),
    ).toEqual([...HISTORY_STATUSES].sort());
  });

  it('sorts running work before waiting work before finished work', () => {
    expect(STATUS_PRIORITY.PROCESSING).toBeLessThan(STATUS_PRIORITY.QUEUED);
    expect(STATUS_PRIORITY.WAITING).toBeLessThan(STATUS_PRIORITY.COMPLETED);
  });
});
