import { beforeEach, describe, expect, it, vi } from 'vitest';

const channel = vi.hoisted(() => ({ appendLine: vi.fn(), show: vi.fn() }));
const createOutputChannel = vi.hoisted(() => vi.fn(() => channel));
vi.mock('vscode', () => ({ window: { createOutputChannel } }));

async function fresh() {
  vi.resetModules();
  return import('./activityLog');
}

beforeEach(() => {
  vi.useFakeTimers();
  vi.setSystemTime(new Date('2026-06-01T12:00:00Z'));
  channel.appendLine.mockReset();
  channel.show.mockReset();
  createOutputChannel.mockClear();
});

describe('activityLog', () => {
  it('creates one lazily-shared output channel', async () => {
    const log = await fresh();
    expect(createOutputChannel).not.toHaveBeenCalled();
    expect(log.getNexusActivityChannel()).toBe(channel);
    log.getNexusActivityChannel();
    log.logNexusActivity('queue', 'x');
    expect(createOutputChannel).toHaveBeenCalledTimes(1);
    expect(createOutputChannel).toHaveBeenCalledWith('Nexus Orchestrator');
  });

  it('timestamps and scopes each line', async () => {
    const log = await fresh();
    log.logNexusActivity('queue', 'submitted');
    expect(channel.appendLine).toHaveBeenCalledWith('[2026-06-01T12:00:00.000Z] [queue] submitted');
  });

  it('shows the log with or without preserving focus', async () => {
    const log = await fresh();
    log.showNexusActivityLog();
    log.showNexusActivityLog(true);
    expect(channel.show).toHaveBeenNthCalledWith(1, false);
    expect(channel.show).toHaveBeenNthCalledWith(2, true);
  });

  it('remembers queued tasks and the most recent one', async () => {
    const log = await fresh();
    expect(log.getActivitySnapshot()).toEqual({ lastQueuedTaskId: undefined });
    expect(log.getKnownTaskSource('a')).toBeUndefined();
    log.rememberQueuedTask('a');
    log.rememberQueuedTask('b');
    expect(log.getKnownTaskSource('a')).toBe('vscode queue');
    expect(log.getKnownTaskSource('b')).toBe('vscode queue');
    expect(log.getKnownTaskSource('c')).toBeUndefined();
    expect(log.getActivitySnapshot()).toEqual({ lastQueuedTaskId: 'b' });
  });
});
