import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NexusClient, Task } from './nexusClient';

vi.mock('vscode', async () => (await import('./test/fakeVscode')).createFakeVscode());
vi.mock('./activityLog', () => ({ getKnownTaskSource: () => undefined }));

import { resetFakeState } from './test/fakeVscode';
import { TaskItem, TaskQueueProvider } from './taskQueueProvider';

const task = (o: Partial<Task> = {}): Task => ({
  id: 'abcd-1234-ef56',
  projectPath: '/workspace',
  targetFile: 'a.ts',
  instruction: 'Do something',
  contextFiles: [],
  status: 'QUEUED',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...o,
});
const client = (tasks: Task[]) =>
  ({ health: vi.fn(), getAllTasks: vi.fn(() => Promise.resolve(tasks)) }) as unknown as NexusClient;

beforeEach(() => resetFakeState());

describe('WAITING tasks in the queue view', () => {
  it('sit with the active tasks, before the finished ones, oldest first', async () => {
    const provider = new TaskQueueProvider(
      client([
        task({ id: 'done', status: 'COMPLETED', updatedAt: '2026-02-01T00:00:00Z' }),
        task({ id: 'wait', status: 'WAITING', createdAt: '2026-01-02T00:00:00Z' }),
        task({ id: 'run', status: 'PROCESSING', createdAt: '2026-01-03T00:00:00Z' }),
      ]),
    );
    const ids = (await provider.getChildren()).map((i) => (i as TaskItem).task.id);
    expect(ids).toEqual(['wait', 'run', 'done']);
  });

  it('say when they resume, in the description and the tooltip', () => {
    const waitUntil = new Date(Date.now() + 3600_000).toISOString();
    const item = new TaskItem(task({ status: 'WAITING', waitUntil }));
    expect(item.description).toMatch(/waits until \d{2}:\d{2}/);
    expect((item.tooltip as { value: string }).value).toContain(`**Resumes:** ${waitUntil}`);
    const plain = new TaskItem(task({ status: 'WAITING' }));
    expect(plain.description).toBe('');
    expect((plain.tooltip as { value: string }).value).not.toContain('Resumes');
  });

  it('can be cancelled like a queued task (same context value)', () => {
    expect(new TaskItem(task({ status: 'WAITING' })).contextValue).toBe('nexusTask');
  });
});
