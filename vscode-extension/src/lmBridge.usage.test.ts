import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const state = {
  parts: [] as string[],
  countTokens: undefined as undefined | ((t: unknown) => Promise<number>),
};

vi.mock('vscode', () => ({
  CancellationTokenSource: class {
    token = {};
    cancel(): void {}
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
        sendRequest: async () => ({
          text: (async function* () {
            for (const p of state.parts) yield p;
          })(),
        }),
        ...(state.countTokens ? { countTokens: state.countTokens } : {}),
      },
    ],
  },
}));

import { startLmBridge, type RunningBridge } from './lmBridge';

describe('lmBridge token usage', () => {
  let bridge: RunningBridge;
  const post = (body: unknown) =>
    fetch(`${bridge.baseUrl}/chat/completions`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${bridge.token}` },
      body: JSON.stringify(body),
    });
  const ask = (stream: boolean) => ({
    model: 'gpt-4o',
    stream,
    messages: [
      { role: 'system', content: 'be brief' },
      { role: 'user', content: 'say hello to me' },
    ],
  });

  beforeEach(async () => {
    state.parts = ['Hel', 'lo'];
    state.countTokens = undefined;
    bridge = await startLmBridge();
  });
  afterEach(async () => {
    await bridge.close();
  });

  it('reports the editor model token counts in a normal response', async () => {
    state.countTokens = async (t: unknown) => (typeof t === 'string' ? 2 : 10);
    await bridge.close();
    bridge = await startLmBridge();
    const body = await (await post(ask(false))).json();
    expect(body.usage).toEqual({ prompt_tokens: 20, completion_tokens: 2, total_tokens: 22 });
  });

  it('reports usage on the final stop chunk of a stream', async () => {
    state.countTokens = async (t: unknown) => (typeof t === 'string' ? 2 : 10);
    await bridge.close();
    bridge = await startLmBridge();
    const text = await (await post(ask(true))).text();
    const chunks = text
      .split('\n\n')
      .filter((e) => e && !e.includes('[DONE]'))
      .map((e) => JSON.parse(e.replace(/^data: /, '')));
    const last = chunks.at(-1);
    expect(last.choices[0].finish_reason).toBe('stop');
    expect(last.usage).toEqual({ prompt_tokens: 20, completion_tokens: 2, total_tokens: 22 });
    expect(chunks.slice(0, -1).every((c) => c.usage === undefined)).toBe(true);
  });

  it('estimates from the text length when the model cannot count tokens', async () => {
    const body = await (await post(ask(false))).json();
    // 'be brief' (8) + 'say hello to me' (15) characters -> 6 tokens; 'Hello' (5) -> 2 tokens.
    expect(body.usage).toEqual({ prompt_tokens: 6, completion_tokens: 2, total_tokens: 8 });
  });

  it('falls back to the estimate when counting throws', async () => {
    state.countTokens = async () => {
      throw new Error('no tokenizer');
    };
    await bridge.close();
    bridge = await startLmBridge();
    const body = await (await post(ask(false))).json();
    expect(body.usage.total_tokens).toBe(8);
  });
});
