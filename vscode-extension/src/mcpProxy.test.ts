import { PassThrough } from 'stream';
import { describe, expect, it } from 'vitest';
import { runProxy } from './mcpProxy';

interface Call {
  url: string;
  init: { method?: string; headers?: Record<string, string>; body?: string };
}

function setup(reply: (body: string) => { status: number; text: string } | Error, token = '') {
  const stdin = new PassThrough();
  const stdout = new PassThrough();
  const stderr = new PassThrough();
  const calls: Call[] = [];
  const done = runProxy({
    stdin,
    stdout,
    stderr,
    url: 'http://127.0.0.1:63988/mcp',
    token,
    fetch: async (url, init) => {
      calls.push({ url, init });
      const r = reply(String(init.body));
      if (r instanceof Error) throw r;
      return { status: r.status, text: async () => r.text };
    },
  });
  const out: string[] = [];
  stdout.on('data', (d: Buffer) => out.push(d.toString()));
  const err: string[] = [];
  stderr.on('data', (d: Buffer) => err.push(d.toString()));
  return { stdin, done, calls, out, err };
}

describe('mcp stdio proxy', () => {
  it('forwards each JSON-RPC line to the daemon and writes the answer as one line', async () => {
    const p = setup((body) => ({
      status: 200,
      text: `{"jsonrpc":"2.0","id":${JSON.parse(body).id},"result":{}}\n`,
    }));
    p.stdin.write(
      '{"jsonrpc":"2.0","id":1,"method":"ping"}\n{"jsonrpc":"2.0","id":2,"method":"ping"}\n',
    );
    p.stdin.end();
    await p.done;
    expect(p.calls.map((c) => c.url)).toEqual([
      'http://127.0.0.1:63988/mcp',
      'http://127.0.0.1:63988/mcp',
    ]);
    expect(p.calls[0]!.init.method).toBe('POST');
    expect(p.calls[0]!.init.headers).toMatchObject({
      'Content-Type': 'application/json',
      Accept: 'application/json',
    });
    expect(p.out.join('')).toBe(
      '{"jsonrpc":"2.0","id":1,"result":{}}\n{"jsonrpc":"2.0","id":2,"result":{}}\n',
    );
  });

  it('sends the token, and says nothing for a notification the daemon answers with 204 or nothing', async () => {
    const p = setup(() => ({ status: 204, text: '' }), 'secret');
    p.stdin.write('{"jsonrpc":"2.0","method":"notifications/initialized"}\n\n   \n');
    p.stdin.end();
    await p.done;
    expect(p.calls).toHaveLength(1); // blank lines are skipped
    expect(p.calls[0]!.init.headers!['Authorization']).toBe('Bearer secret');
    expect(p.out.join('')).toBe('');
  });

  it('answers a request with a JSON-RPC error when the daemon cannot be reached, and keeps going', async () => {
    let n = 0;
    const p = setup(() =>
      ++n === 1
        ? new Error('ECONNREFUSED')
        : { status: 200, text: '{"jsonrpc":"2.0","id":2,"result":{}}' },
    );
    p.stdin.write(
      '{"jsonrpc":"2.0","id":1,"method":"tools/list"}\n{"jsonrpc":"2.0","id":2,"method":"ping"}\n',
    );
    p.stdin.end();
    await p.done;
    const lines = p.out
      .join('')
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l));
    expect(lines[0]).toMatchObject({ id: 1, error: { code: -32000 } });
    expect(lines[0].error.message).toContain('ECONNREFUSED');
    expect(lines[1]).toMatchObject({ id: 2, result: {} });
    expect(p.err.join('')).toContain('ECONNREFUSED');
  });

  it('does not answer a notification it could not deliver, and reports a bad HTTP status', async () => {
    const p = setup((body) =>
      body.includes('notifications') ? new Error('down') : { status: 500, text: 'boom' },
    );
    p.stdin.write(
      '{"jsonrpc":"2.0","method":"notifications/x"}\n{"jsonrpc":"2.0","id":7,"method":"tools/call"}\nnot json\n',
    );
    p.stdin.end();
    await p.done;
    const lines = p.out
      .join('')
      .trim()
      .split('\n')
      .map((l) => JSON.parse(l));
    expect(lines).toHaveLength(2); // the notification gets no answer
    expect(lines[0]).toMatchObject({
      id: 7,
      error: { message: expect.stringContaining('HTTP 500') },
    });
    expect(lines[1]).toMatchObject({ id: null, error: { code: -32700 } });
  });

  it('stays quiet about a failed notification, and copes with a rejection that is not an Error', async () => {
    const p = setup((body) =>
      body.includes('notifications')
        ? { status: 500, text: 'x' }
        : (Object.assign(new Error('x'), {}) as Error),
    );
    p.stdin.write('{"jsonrpc":"2.0","method":"notifications/y"}\n');
    p.stdin.end();
    await p.done;
    expect(p.out.join('')).toBe('');

    const stdin = new PassThrough();
    const stdout = new PassThrough();
    const seen: string[] = [];
    stdout.on('data', (d: Buffer) => seen.push(d.toString()));
    const done = runProxy({
      stdin,
      stdout,
      stderr: new PassThrough(),
      url: 'u',
      token: '',
      fetch: () => Promise.reject('plain string'),
    });
    stdin.write('{"jsonrpc":"2.0","id":3,"method":"ping"}\n');
    stdin.end();
    await done;
    expect(seen.join('')).toContain('plain string');
  });
});
