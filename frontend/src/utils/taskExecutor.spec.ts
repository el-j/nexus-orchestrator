import { describe, expect, it } from 'vitest';
import { whoRanTask } from './taskExecutor';
import type { Task } from '../types/domain';

const task = (o: Partial<Task>): Task => ({ status: 'QUEUED', ...o }) as Task;

describe('whoRanTask', () => {
  it('names the provider when Nexus ran the task', () => {
    expect(whoRanTask(task({ status: 'COMPLETED', ranBy: 'Ollama' }))).toEqual({
      text: 'Run by Nexus on Ollama',
      byNexus: true,
    });
    expect(whoRanTask(task({ status: 'FAILED', ranBy: 'Ollama' })).byNexus).toBe(true);
  });

  it('says plainly when an agent set the status and Nexus did nothing', () => {
    for (const status of ['PROCESSING', 'COMPLETED', 'FAILED'] as const) {
      expect(whoRanTask(task({ status }))).toEqual({
        text: 'Status set by an agent — Nexus did not run this task',
        byNexus: false,
      });
    }
  });

  it('describes the other states without claiming anything was run', () => {
    expect(whoRanTask(task({ status: 'QUEUED' })).text).toBe('Waiting for a provider');
    expect(whoRanTask(task({ status: 'DRAFT' })).text).toBe('Not started');
    expect(whoRanTask(task({ status: 'BACKLOG' })).text).toBe('Not started');
    expect(whoRanTask(task({ status: 'CANCELLED' })).text).toBe('');
    expect(whoRanTask(task({ status: 'NO_PROVIDER' })).text).toBe('');
  });
});
