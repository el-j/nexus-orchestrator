import { describe, it, expect, vi, beforeEach } from 'vitest';

type Handler = (e: { type: string; [k: string]: unknown }) => void;
const sse = vi.hoisted(() => ({ handlers: [] as Handler[], connect: vi.fn(), on: vi.fn() }));
vi.mock('./useGlobalSSE', () => ({
  useGlobalSSE: () => ({
    on: (type: string, h: Handler) => {
      sse.on(type);
      sse.handlers.push(h);
    },
    connect: sse.connect,
  }),
}));

let live: typeof import('./useLiveOutput');
const send = (e: { type: string; [k: string]: unknown }) => sse.handlers.forEach((h) => h(e));

beforeEach(async () => {
  vi.resetModules();
  sse.handlers.length = 0;
  sse.on.mockClear();
  sse.connect.mockClear();
  live = await import('./useLiveOutput');
});

describe('useLiveOutput', () => {
  it('listens once, to every event, and makes sure the stream is connected', () => {
    live.useLiveOutput();
    live.useLiveOutput();
    expect(sse.on).toHaveBeenCalledTimes(1);
    expect(sse.on).toHaveBeenCalledWith('*');
    expect(sse.connect).toHaveBeenCalledTimes(1);
  });

  it('appends chunks per task', () => {
    const { liveText } = live.useLiveOutput();
    send({ type: 'task.chunk', taskId: 'a', text: 'Hel' });
    send({ type: 'task.chunk', taskId: 'a', text: 'lo' });
    send({ type: 'task.chunk', taskId: 'b', text: 'other' });
    expect(liveText('a')).toBe('Hello');
    expect(liveText('b')).toBe('other');
    expect(liveText('nobody')).toBe('');
  });

  it('clears on restart', () => {
    const { liveText } = live.useLiveOutput();
    send({ type: 'task.chunk', taskId: 'a', text: 'partial' });
    send({ type: 'task.restart', taskId: 'a' });
    expect(liveText('a')).toBe('');
    send({ type: 'task.chunk', taskId: 'a', text: 'fresh' });
    expect(liveText('a')).toBe('fresh');
  });

  it('keeps only the newest characters', () => {
    const { liveText } = live.useLiveOutput();
    send({ type: 'task.chunk', taskId: 'a', text: 'x'.repeat(live.MAX_LIVE_CHARS) });
    send({ type: 'task.chunk', taskId: 'a', text: 'END' });
    expect(liveText('a')).toHaveLength(live.MAX_LIVE_CHARS);
    expect(liveText('a').endsWith('END')).toBe(true);
  });

  it.each([
    'task.completed',
    'task.failed',
    'task.cancelled',
    'task.too_large',
    'task.no_provider',
    'task.queued',
  ])('drops the text on %s', (type) => {
    const { liveText } = live.useLiveOutput();
    send({ type: 'task.chunk', taskId: 'a', text: 'abc' });
    send({ type, taskId: 'a' });
    expect(liveText('a')).toBe('');
  });

  it('ignores events without a task id, chunks without text and unrelated types', () => {
    const { liveText } = live.useLiveOutput();
    send({ type: 'task.chunk', text: 'orphan' });
    send({ type: 'task.chunk', taskId: 'a' });
    send({ type: 'task.processing', taskId: 'a' });
    send({ type: 'log', taskId: 'a', text: 'x' });
    expect(liveText('a')).toBe('');
  });
});
