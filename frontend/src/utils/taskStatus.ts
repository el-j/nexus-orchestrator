import type { TaskStatus } from '../types/domain';

/** The finished states a task history lists. */
export const HISTORY_STATUSES: ReadonlySet<TaskStatus> = new Set<TaskStatus>([
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'TOO_LARGE',
  'NO_PROVIDER',
]);

/** A value of the history filter bar: every finished state, or all of them. */
export type HistoryFilterValue =
  'ALL' | 'COMPLETED' | 'FAILED' | 'CANCELLED' | 'TOO_LARGE' | 'NO_PROVIDER';

/** The buttons of the history filter bar, in display order. */
export const HISTORY_FILTERS: { label: string; value: HistoryFilterValue }[] = [
  { label: 'All', value: 'ALL' },
  { label: 'Completed', value: 'COMPLETED' },
  { label: 'Failed', value: 'FAILED' },
  { label: 'Cancelled', value: 'CANCELLED' },
  { label: 'Too Large', value: 'TOO_LARGE' },
  { label: 'No Provider', value: 'NO_PROVIDER' },
];

/** Sort order of the work list: running first, then waiting, drafts, and finished states last. */
export const STATUS_PRIORITY: Record<string, number> = {
  PROCESSING: 0,
  QUEUED: 1,
  WAITING: 2,
  DRAFT: 3,
  BACKLOG: 4,
  FAILED: 5,
  COMPLETED: 6,
  CANCELLED: 7,
  TOO_LARGE: 8,
  NO_PROVIDER: 9,
};
