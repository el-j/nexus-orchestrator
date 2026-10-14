import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const cancelled: boolean[] = [];
const state = {
  models: [] as Array<{ id: string; vendor: string; sendRequest: ReturnType<typeof vi.fn> }>,
};

vi.mock('vscode', () => ({
  CancellationTokenSource: class {
    token = { id: 'token' };
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
    selectChatModels: async (sel: { id?: string }) =>
      state.models.filter((m) => sel.id === undefined || m.id === sel.id),
  },
}));

import { startLmBridge, type RunningBridge } from './lmBridge';

function reply(...parts: string[]): ReturnType<typeof vi.fn> {
  return vi.fn(async () => ({
    text: (async function* () {
      for (const p of parts) yield p;
    })(),
  }));
}

describe('lmBridge', () => {
  let bridge: RunningBridge;
  const call = (path: string, init: RequestInit = {}, token: string | null = bridge.token) =>
    fetch(`${bridge.baseUrl.replace('/v1', '')}${path}`, {
      ...init,
      headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...init.headers },
    });
  const chat = (body: unknown) =>
    call('/v1/chat/completions', { method: 'POST', body: JSON.stringify(body) });

  beforeEach(async () => {
    cancelled.length = 0;
    state.models = [{ id: 'gpt-4o', vendor: 'copilot', sendRequest: reply('Hel', 'lo') }];
    bridge = await startLmBridge();
  });
  afterEach(async () => {
    await bridge.close();
  });

  it('listens on loopback only and ends the base URL in /v1', () => {
    expect(bridge.baseUrl).toMatch(/^http:\/\/127\.0\.0\.1:\d+\/v1$/);
    expect(bridge.token).toHaveLength(48);
  });

  it('refuses a missing or wrong token', async () => {
    expect((await call('/v1/models', {}, null)).status).toBe(401);
    expect((await call('/v1/models', {}, 'nope')).status).toBe(401);
    expect(state.models[0]!.sendRequest).not.toHaveBeenCalled();
  });

  it('returns 404 for an unknown route', async () => {
    expect((await call('/v1/other')).status).toBe(404);
  });

  it('lists the editor models in the OpenAI shape', async () => {
    const body = await (await call('/v1/models')).json();
    expect(body).toEqual({
      object: 'list',
      data: [{ id: 'gpt-4o', object: 'model', owned_by: 'copilot' }],
    });
  });

  it('answers a chat completion with the joined streamed text', async () => {
    const resp = await chat({ model: 'gpt-4o', messages: [{ role: 'user', content: 'hi' }] });
    expect(resp.status).toBe(200);
    const body = (await resp.json()) as { choices: Array<{ message: { content: string } }> };
    expect(body.choices[0]!.message.content).toBe('Hello');
  });

  it('sends system and user as user turns, keeps assistant, and stringifies odd content', async () => {
    await chat({
      model: 'gpt-4o',
      messages: [
        { role: 'system', content: 'rules' },
        { role: 'assistant', content: 'ok' },
        { role: 'user', content: [{ type: 'text', text: 'x' }] },
        { role: 'user' },
      ],
    });
    const sent = state.models[0]!.sendRequest.mock.calls[0]![0];
    expect(sent).toEqual([
      { role: 'user', text: 'rules' },
      { role: 'assistant', text: 'ok' },
      { role: 'user', text: '[{"type":"text","text":"x"}]' },
      { role: 'user', text: '""' },
    ]);
  });

  it.each([
    ['no model', { messages: [{ role: 'user', content: 'x' }] }],
    ['model not a string', { model: 5, messages: [{ role: 'user', content: 'x' }] }],
    ['no messages', { model: 'gpt-4o' }],
    ['empty messages', { model: 'gpt-4o', messages: [] }],
    ['messages not an array', { model: 'gpt-4o', messages: 'x' }],
  ])('rejects %s with 400', async (_name, body) => {
    expect((await chat(body)).status).toBe(400);
  });

  it('rejects a body that is not a JSON object', async () => {
    for (const raw of ['not json', '[1]', 'null']) {
      const resp = await call('/v1/chat/completions', { method: 'POST', body: raw });
      expect(resp.status).toBe(400);
    }
  });

  it('rejects an oversized body with 413', async () => {
    const big = JSON.stringify({ model: 'gpt-4o', messages: [], pad: 'x'.repeat(9 * 1024 * 1024) });
    const resp = await call('/v1/chat/completions', { method: 'POST', body: big }).catch(
      () => null,
    );
    // The server may reset the connection once the limit is hit; a response must say 413.
    if (resp) expect(resp.status).toBe(413);
  });

  it('returns 404 for a model this editor does not have', async () => {
    const resp = await chat({ model: 'ghost', messages: [{ role: 'user', content: 'x' }] });
    expect(resp.status).toBe(404);
  });

  it.each([
    ['NoPermissions', 'consent refused', 403],
    ['NotFound', 'gone', 404],
    ['Blocked', 'filtered', 429],
    ['', 'monthly quota exceeded', 429],
    ['', 'rate limit', 429],
    ['', 'boom', 502],
  ])('maps a model failure %s/%s to HTTP %i', async (code, message, status) => {
    state.models[0]!.sendRequest.mockRejectedValue(Object.assign(new Error(message), { code }));
    const resp = await chat({ model: 'gpt-4o', messages: [{ role: 'user', content: 'x' }] });
    expect(resp.status).toBe(status);
    expect(((await resp.json()) as { error: { message: string } }).error.message).toBe(message);
  });

  it('copes with a failure that is not an Error', async () => {
    state.models[0]!.sendRequest.mockRejectedValue('plain string');
    expect(
      (await chat({ model: 'gpt-4o', messages: [{ role: 'user', content: 'x' }] })).status,
    ).toBe(502);
    state.models[0]!.sendRequest.mockRejectedValue(null);
    expect(
      (await chat({ model: 'gpt-4o', messages: [{ role: 'user', content: 'x' }] })).status,
    ).toBe(502);
  });

  it('cancels the model request when the caller disconnects', async () => {
    let release: () => void = () => {};
    state.models[0]!.sendRequest.mockImplementation(
      () =>
        new Promise((resolve) => (release = () => resolve({ text: (async function* () {})() }))),
    );
    const ac = new AbortController();
    const pending = call('/v1/chat/completions', {
      method: 'POST',
      body: JSON.stringify({ model: 'gpt-4o', messages: [{ role: 'user', content: 'x' }] }),
      signal: ac.signal,
    }).catch(() => null);
    await vi.waitFor(() => expect(state.models[0]!.sendRequest).toHaveBeenCalled());
    ac.abort();
    await pending;
    await vi.waitFor(() => expect(cancelled.length).toBeGreaterThan(0));
    release();
  });

  it('cancels requests in flight when it is closed', async () => {
    state.models[0]!.sendRequest.mockImplementation(() => new Promise(() => {}));
    void call('/v1/chat/completions', {
      method: 'POST',
      body: JSON.stringify({ model: 'gpt-4o', messages: [{ role: 'user', content: 'x' }] }),
    }).catch(() => null);
    await vi.waitFor(() => expect(state.models[0]!.sendRequest).toHaveBeenCalled());
    await bridge.close();
    expect(cancelled.length).toBeGreaterThan(0);
    bridge = await startLmBridge(); // afterEach closes a live one
  });
});
