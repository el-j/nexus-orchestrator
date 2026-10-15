/**
 * taskExecutor.ts: says who ran a task. Nexus runs a task itself on one of its providers (the
 * engine then writes "completed via <provider>" into the log, which the daemon reports as
 * `ranBy`); an agent working on its own can also mark a task processing or done, and then Nexus
 * ran nothing. The UI must not blur the two.
 */
import type { Task } from '../types/domain';

/** What to show about the executor of a task; `byNexus` is true when Nexus ran it. */
export interface TaskExecutor {
  text: string;
  byNexus: boolean;
}

/** Describes who ran (or is running, or will run) a task. */
export function whoRanTask(task: Task): TaskExecutor {
  if (task.ranBy) return { text: `Run by Nexus on ${task.ranBy}`, byNexus: true };
  switch (task.status) {
    case 'PROCESSING':
    case 'COMPLETED':
    case 'FAILED':
      return { text: 'Status set by an agent — Nexus did not run this task', byNexus: false };
    case 'QUEUED':
      return { text: 'Waiting for a provider', byNexus: false };
    case 'DRAFT':
    case 'BACKLOG':
      return { text: 'Not started', byNexus: false };
    default:
      return { text: '', byNexus: false };
  }
}
