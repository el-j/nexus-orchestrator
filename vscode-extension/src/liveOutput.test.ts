import { beforeEach, describe, expect, it, vi } from 'vitest';

const h = vi.hoisted(() => ({
  out: [] as string[],
  channel: undefined as unknown,
  enabled: true,
  commandHandler: undefined as undefined | (() => void),
}));

vi.mock('vscode', () => ({
  window: {
    createOutputChannel: () => {
      h.channel = {
        append: (s: string) => h.out.push(s),
        appendLine: (s: string) => h.out.push(s + '\n'),
        show: vi.fn(),
        dispose: vi.fn(),
      };
      return h.channel;
    },
  },
  commands: {
    registerCommand: (_id: string, fn: () => void) => {
      h.commandHandler = fn;
      return { dispose() {} };
    },
  },
  workspace: {
    getConfiguration: () => ({ get: (_k: string, d: boolean) => (h.enabled ? d : false) }),
  },
}));

import { LiveOutput } from './liveOutput';
import type { NexusClient } from './nexusClient';

const sse = (...events: object[]) =>
  new Response(events.map((e) => `data: ${JSON.stringify(e)}\n\n`).join(''));
const printed = () => h.out.join('');

describe('LiveOutput', () => {
  let openEvents: ReturnType<typeof vi.fn>;
  let live: LiveOutput;
  beforeEach(() => {
    h.out.length = 0;
    h.enabled = true;
    openEvents = vi.fn();
    live = new LiveOutput({ openEvents } as unknown as NexusClient);
  });

  /** Feeds one stream through the reader and stops the loop afterwards. */
  async function play(resp: Response): Promise<void> {
    openEvents.mockResolvedValueOnce(resp).mockImplementation(async () => {
      live.dispose();
      throw new Error('stop');
    });
    vi.useFakeTimers();
    const done = live.follow();
    await vi.advanceTimersByTimeAsync(10_000); // read the stream, wait out the reconnect pause
    await done;
    vi.useRealTimers();
  }

  it('prints a task header once, then the chunks', async () => {
    await play(
      sse(
        { type: 'task.chunk', taskId: 'abcdef123456', text: 'Hel' },
        { type: 'task.chunk', taskId: 'abcdef123456', text: 'lo' },
      ),
    );
    expect(printed()).toBe('\n[abcdef12] Hello');
  });

  it('prints a new header when the text switches between tasks', async () => {
    await play(
      sse(
        { type: 'task.chunk', taskId: 'aaaaaaaa1', text: 'A1' },
        { type: 'task.chunk', taskId: 'bbbbbbbb2', text: 'B1' },
        { type: 'task.chunk', taskId: 'aaaaaaaa1', text: 'A2' },
      ),
    );
    expect(printed()).toBe('\n[aaaaaaaa] A1\n[bbbbbbbb] B1\n[aaaaaaaa] A2');
  });

  it('marks a restart only for a task whose text was shown, and reprints the header', async () => {
    await play(
      sse(
        { type: 'task.restart', taskId: 'cccccccc1' },
        { type: 'task.chunk', taskId: 'cccccccc1', text: 'partial' },
        { type: 'task.restart', taskId: 'cccccccc1' },
        { type: 'task.chunk', taskId: 'cccccccc1', text: 'fresh' },
      ),
    );
    expect(printed()).toBe('\n[cccccccc] partial\n[cccccccc] ↻ new attempt\n\n[cccccccc] fresh');
  });

  it('closes a shown task with its outcome, once', async () => {
    await play(
      sse(
        { type: 'task.chunk', taskId: 'dddddddd1', text: 'x' },
        { type: 'task.completed', taskId: 'dddddddd1' },
        { type: 'task.completed', taskId: 'dddddddd1' },
        { type: 'task.failed', taskId: 'neverseen1' },
      ),
    );
    expect(printed()).toBe('\n[dddddddd] x\n[dddddddd] completed\n');
  });

  it('ignores events it cannot use', async () => {
    await play(
      sse(
        { type: 'task.chunk', text: 'no task' },
        { taskId: 'eeeeeeee1', text: 'no type' },
        { type: 'task.chunk', taskId: 'eeeeeeee1' },
        { type: 'task.processing', taskId: 'eeeeeeee1' },
      ),
    );
    expect(printed()).toBe('');
  });

  it('skips malformed frames and other lines, and joins a frame split across reads', async () => {
    const enc = new TextEncoder();
    const body = new ReadableStream({
      start(c) {
        c.enqueue(enc.encode(': keep-alive\n\ndata: {broken\n\ndata: {"type":"task.chunk","taskI'));
        c.enqueue(enc.encode('d":"ffffffff1","text":"ok"}\n\n'));
        c.close();
      },
    });
    await play(new Response(body));
    expect(printed()).toBe('\n[ffffffff] ok');
  });

  describe('reconnecting', () => {
    beforeEach(() => vi.useFakeTimers());

    it('retries after a failure and after a closed stream, waiting between tries', async () => {
      openEvents
        .mockRejectedValueOnce(new Error('daemon away'))
        .mockResolvedValueOnce(sse({ type: 'task.chunk', taskId: 'gggggggg1', text: 'back' }))
        .mockImplementation(async () => {
          live.dispose();
          throw new Error('stop');
        });
      const done = live.follow();
      await vi.advanceTimersByTimeAsync(0);
      expect(openEvents).toHaveBeenCalledTimes(1); // waiting before the second try
      await vi.advanceTimersByTimeAsync(5_000);
      expect(printed()).toContain('back');
      await vi.advanceTimersByTimeAsync(5_000);
      await done;
      expect(openEvents).toHaveBeenCalledTimes(3);
      vi.useRealTimers();
    });

    it('stops when disposed during the wait', async () => {
      openEvents.mockRejectedValue(new Error('away'));
      const done = live.follow();
      await vi.advanceTimersByTimeAsync(0);
      live.dispose();
      await vi.advanceTimersByTimeAsync(5_000);
      await done;
      expect(openEvents).toHaveBeenCalledTimes(1);
      vi.useRealTimers();
    });
  });

  describe('register', () => {
    const ctx = () => ({ subscriptions: [] as unknown[] });
    it('follows the stream, adds the command and is disposed with the context', () => {
      openEvents.mockReturnValue(new Promise(() => {}));
      const c = ctx();
      live.register(c as never);
      expect(openEvents).toHaveBeenCalledTimes(1);
      expect(c.subscriptions).toContain(live);
      h.commandHandler!();
      expect((h.channel as { show: ReturnType<typeof vi.fn> }).show).toHaveBeenCalledWith(true);
    });
    it('does not follow when switched off, but still offers the command', () => {
      h.enabled = false;
      live.register(ctx() as never);
      expect(openEvents).not.toHaveBeenCalled();
      expect(h.commandHandler).toBeDefined();
    });
  });

  it('aborts the stream and closes the channel on dispose', () => {
    live.dispose();
    expect((h.channel as { dispose: ReturnType<typeof vi.fn> }).dispose).toHaveBeenCalled();
  });
});
