import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cancelled: boolean[] = [];
const state = {
  parts: (async function* () {})() as AsyncIterable<string>,
  sendRequest: vi.fn(),
};

vi.mock('vscode', () => ({
  CancellationTokenSource: class {
    token = {};
    cancel(): void {
      cancelled.push(true);
    }
    dispose(): void {}
  },
  LanguageModelChatMessage: {
    User: (text: string) => ({ role: 'user', text }),
    Assistant: (text: string) => ({ role: 'assistant', text }),
  },
  lm: {
    selectChatModels: async () => [
      {
        id: 'gpt-4o',
        vendor: 'copilot',
        sendRequest: (...a: unknown[]) => state.sendRequest(...a),
      },
    ],
  },
}));

import { startLmBridge, type RunningBridge } from './lmBridge';

/** Parses an SSE body into its `data:` payloads. */
function events(body: string): string[] {
  return body
    .split('\n\n')
    .filter(Boolean)
    .map((e) => e.replace(/^data: /, ''));
}

describe('lmBridge streaming', () => {
  let bridge: RunningBridge;
  const post = (body: unknown, signal?: AbortSignal) =>
    fetch(`${bridge.baseUrl}/chat/completions`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${bridge.token}` },
      body: JSON.stringify(body),
      signal,
    });
  const ask = { model: 'gpt-4o', stream: true, messages: [{ role: 'user', content: 'hi' }] };

  beforeEach(async () => {
    cancelled.length = 0;
    state.sendRequest = vi.fn(async () => ({ text: state.parts }));
    bridge = await startLmBridge();
  });
  afterEach(async () => {
    await bridge.close();
  });

  it('streams OpenAI chunks: role, one delta per part, a stop chunk and [DONE]', async () => {
    state.parts = (async function* () {
      yield 'Hel';
      yield 'lo';
    })();
    const resp = await post(ask);
    expect(resp.status).toBe(200);
    expect(resp.headers.get('content-type')).toBe('text/event-stream');
    const e = events(await resp.text());
    expect(e.at(-1)).toBe('[DONE]');
    const chunks = e.slice(0, -1).map((x) => JSON.parse(x));
    expect(chunks.map((c) => c.choices[0].delta)).toEqual([
      { role: 'assistant' },
      { content: 'Hel' },
      { content: 'lo' },
      {},
    ]);
    expect(chunks.map((c) => c.choices[0].finish_reason)).toEqual([null, null, null, 'stop']);
    expect(chunks[0].object).toBe('chat.completion.chunk');
    expect(chunks[0].model).toBe('gpt-4o');
  });

  it('delivers the first part before the model has finished', async () => {
    let finish: () => void = () => {};
    const gate = new Promise<void>((r) => (finish = r));
    state.parts = (async function* () {
      yield 'first';
      await gate;
      yield 'second';
    })();
    const resp = await post(ask);
    const reader = resp.body!.getReader();
    const dec = new TextDecoder();
    let seen = '';
    while (!seen.includes('first')) seen += dec.decode((await reader.read()).value);
    expect(seen).not.toContain('second');
    finish();
    let rest = '';
    for (let r = await reader.read(); !r.done; r = await reader.read()) rest += dec.decode(r.value);
    expect(rest).toContain('second');
    expect(rest).toContain('[DONE]');
  });

  it('reports a failure before the first byte as a normal error response', async () => {
    state.sendRequest.mockRejectedValue(
      Object.assign(new Error('no consent'), { code: 'NoPermissions' }),
    );
    const resp = await post(ask);
    expect(resp.status).toBe(403);
    expect(resp.headers.get('content-type')).toBe('application/json');
  });

  it('reports a failure mid-stream as an error event and no [DONE]', async () => {
    state.parts = (async function* () {
      yield 'partial';
      throw new Error('quota exceeded');
    })();
    const resp = await post(ask);
    expect(resp.status).toBe(200);
    const e = events(await resp.text());
    expect(e).not.toContain('[DONE]');
    // The status lets Nexus classify the failure (429 = rate limited) and put the provider on cooldown.
    expect(JSON.parse(e.at(-1)!)).toEqual({ error: { message: 'quota exceeded', status: 429 } });
  });

  it('cancels the model request when the caller disconnects mid-stream', async () => {
    let release: () => void = () => {};
    state.parts = (async function* () {
      yield 'first';
      await new Promise<void>((r) => (release = r));
    })();
    const ac = new AbortController();
    const resp = await post(ask, ac.signal);
    await resp.body!.getReader().read();
    ac.abort();
    await vi.waitFor(() => expect(cancelled.length).toBeGreaterThan(0));
    release();
  });

  it('still answers whole when stream is false or absent', async () => {
    state.parts = (async function* () {
      yield 'whole';
    })();
    for (const stream of [false, undefined, 'true']) {
      state.parts = (async function* () {
        yield 'whole';
      })();
      const resp = await post({ ...ask, stream });
      expect(resp.headers.get('content-type')).toBe('application/json');
      const body = (await resp.json()) as { choices: Array<{ message: { content: string } }> };
      expect(body.choices[0]!.message.content).toBe('whole');
    }
  });
});
