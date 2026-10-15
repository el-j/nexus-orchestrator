import { EventEmitter } from 'events';
import * as https from 'https';
import * as core from '@actions/core';
import {
  httpsGet,
  fetchAgent,
  fetchCategoryIndex,
  resolveAgents,
  resolveCategory,
  AGENT_CATEGORIES,
} from '../src/agents';

jest.mock('https');

const warnings = (core as unknown as { _warnings: string[] })._warnings;

interface Reply {
  status?: number;
  headers?: Record<string, string>;
  body?: string;
  error?: Error;
  timeout?: boolean;
  resEvent?: 'error';
  noStatus?: boolean;
}
type Route = Reply | ((url: string, opts: https.RequestOptions) => Reply);

const calls: Array<{ url: string; opts: https.RequestOptions }> = [];
let routes: Array<{ match: RegExp; reply: Route }> = [];

function install(): void {
  (https.get as unknown as jest.Mock).mockImplementation(
    (url: string, opts: https.RequestOptions, cb: (res: EventEmitter) => void) => {
      calls.push({ url, opts });
      const req = new EventEmitter() as EventEmitter & { destroy: jest.Mock };
      req.destroy = jest.fn((err?: Error) => {
        if (err) setImmediate(() => req.emit('error', err));
      });
      const route = routes.find((r) => r.match.test(url));
      const reply: Reply = route
        ? typeof route.reply === 'function'
          ? route.reply(url, opts)
          : route.reply
        : { status: 404, body: '' };
      setImmediate(() => {
        if (reply.error) {
          req.emit('error', reply.error);
          return;
        }
        if (reply.timeout) {
          req.emit('timeout');
          return;
        }
        const res = new EventEmitter() as EventEmitter & {
          statusCode?: number | undefined;
          headers: Record<string, string>;
          resume: jest.Mock;
        };
        res.statusCode = reply.noStatus ? undefined : (reply.status ?? 200);
        res.headers = reply.headers ?? {};
        res.resume = jest.fn();
        cb(res);
        if (reply.resEvent === 'error') {
          res.emit('error', new Error('stream broke'));
          return;
        }
        if (reply.body !== undefined) res.emit('data', Buffer.from(reply.body));
        res.emit('end');
      });
      return req;
    },
  );
}

const agentMd = (name: string, extra = '') =>
  `---\nname: ${name}\ndescription: Does ${name}\n${extra}---\nYou are ${name}.`;

beforeEach(() => {
  calls.length = 0;
  routes = [];
  warnings.length = 0;
  delete process.env['GITHUB_TOKEN'];
  install();
});

describe('httpsGet', () => {
  it('returns the body of a 200', async () => {
    routes = [{ match: /./, reply: { body: 'hello' } }];
    await expect(httpsGet('https://x/a')).resolves.toBe('hello');
  });

  it('rejects with the status for anything else, and "?" for a missing status', async () => {
    routes = [{ match: /404/, reply: { status: 404 } }];
    await expect(httpsGet('https://x/404')).rejects.toThrow('HTTP 404 for https://x/404');
    routes = [{ match: /./, reply: { status: 0 } }];
    await expect(httpsGet('https://x/none')).rejects.toThrow('HTTP ? for');
  });

  it('treats a response without any status as a failure', async () => {
    routes = [{ match: /./, reply: { noStatus: true } }];
    await expect(httpsGet('https://x/a')).rejects.toThrow('HTTP ? for');
  });

  it('follows absolute and relative redirects', async () => {
    routes = [
      { match: /\/start$/, reply: { status: 302, headers: { location: 'https://x/mid' } } },
      { match: /\/mid$/, reply: { status: 301, headers: { location: '/final' } } },
      { match: /\/final$/, reply: { body: 'done' } },
    ];
    await expect(httpsGet('https://x/start')).resolves.toBe('done');
  });

  it('treats a redirect without a Location as the final response', async () => {
    routes = [{ match: /./, reply: { status: 302 } }];
    await expect(httpsGet('https://x/a')).rejects.toThrow('HTTP 302');
  });

  it('gives up on a redirect loop instead of recursing forever', async () => {
    routes = [{ match: /./, reply: { status: 302, headers: { location: 'https://x/loop' } } }];
    await expect(httpsGet('https://x/loop')).rejects.toThrow('Too many redirects');
    expect(calls.length).toBe(6); // the original + 5 redirects
  });

  it('does not forward request headers to a different host', async () => {
    process.env['GITHUB_TOKEN'] = 'tok';
    routes = [
      {
        match: /api\.github\.com/,
        reply: { status: 302, headers: { location: 'https://evil.example/x' } },
      },
      { match: /evil\.example/, reply: { body: '[]' } },
    ];
    await fetchCategoryIndex('design');
    expect(calls[0].opts.headers).toMatchObject({ Authorization: 'Bearer tok' });
    expect(calls[1].opts.headers).toEqual({});
  });

  it('keeps headers on a same-host redirect', async () => {
    process.env['GITHUB_TOKEN'] = 'tok';
    routes = [
      { match: /contents\/design/, reply: { status: 302, headers: { location: '/moved' } } },
      { match: /moved/, reply: { body: '[]' } },
    ];
    await fetchCategoryIndex('design');
    expect(calls[1].opts.headers).toMatchObject({ Authorization: 'Bearer tok' });
  });

  it('rejects on a network error and on a stream error', async () => {
    routes = [{ match: /./, reply: { error: new Error('ECONNRESET') } }];
    await expect(httpsGet('https://x/a')).rejects.toThrow('ECONNRESET');
    routes = [{ match: /./, reply: { resEvent: 'error' } }];
    await expect(httpsGet('https://x/a')).rejects.toThrow('stream broke');
  });

  it('destroys the request on timeout so the socket does not linger', async () => {
    routes = [{ match: /./, reply: { timeout: true } }];
    await expect(httpsGet('https://x/slow')).rejects.toThrow('Request timed out: https://x/slow');
  });
});

describe('fetchAgent', () => {
  it('parses an agent file with a default colour', async () => {
    routes = [
      {
        match: /raw\.githubusercontent\.com.*\/design\/a\.md/,
        reply: { body: agentMd('Pixel Pro') },
      },
    ];
    await expect(fetchAgent('design', 'a.md')).resolves.toEqual({
      name: 'Pixel Pro',
      slug: 'pixel-pro',
      description: 'Does Pixel Pro',
      color: 'gray',
      category: 'design',
      systemPrompt: 'You are Pixel Pro.',
    });
    expect(calls[0].url).toBe(
      'https://raw.githubusercontent.com/el-j/agency-agents/main/design/a.md',
    );
  });

  it('uses the colour and ref it is given', async () => {
    routes = [{ match: /./, reply: { body: agentMd('X', 'color: red\n') } }];
    const a = await fetchAgent('design', 'a.md', 'v2');
    expect(a?.color).toBe('red');
    expect(calls[0].url).toContain('/v2/design/a.md');
  });

  it('returns null for a missing file, a network failure, or incomplete frontmatter', async () => {
    expect(await fetchAgent('design', 'gone.md')).toBeNull();
    routes = [{ match: /./, reply: { error: new Error('boom') } }];
    expect(await fetchAgent('design', 'a.md')).toBeNull();
    routes = [{ match: /./, reply: { body: '---\nname: Only Name\n---\nbody' } }];
    expect(await fetchAgent('design', 'a.md')).toBeNull();
    routes = [{ match: /./, reply: { body: 'no frontmatter' } }];
    expect(await fetchAgent('design', 'a.md')).toBeNull();
  });
});

describe('fetchCategoryIndex', () => {
  const listing = JSON.stringify([
    { name: 'a.md', type: 'file' },
    { name: 'README.txt', type: 'file' },
    { name: 'sub', type: 'dir' },
    { name: 'b.md', type: 'file' },
  ]);

  it('lists only markdown files', async () => {
    routes = [{ match: /./, reply: { body: listing } }];
    await expect(fetchCategoryIndex('design')).resolves.toEqual(['a.md', 'b.md']);
    expect(calls[0].url).toBe(
      'https://api.github.com/repos/el-j/agency-agents/contents/design?ref=main',
    );
  });

  it('identifies itself (GitHub rejects requests without a User-Agent)', async () => {
    routes = [{ match: /./, reply: { body: '[]' } }];
    await fetchCategoryIndex('design');
    expect(calls[0].opts.headers).toMatchObject({
      'User-Agent': 'nexus-orchestrator-action',
      Accept: 'application/vnd.github+json',
    });
    expect(calls[0].opts.headers).not.toHaveProperty('Authorization');
  });

  it('authenticates with GITHUB_TOKEN when set', async () => {
    process.env['GITHUB_TOKEN'] = 'abc';
    routes = [{ match: /./, reply: { body: '[]' } }];
    await fetchCategoryIndex('design', 'dev');
    expect(calls[0].opts.headers).toMatchObject({ Authorization: 'Bearer abc' });
    expect(calls[0].url).toContain('ref=dev');
  });

  it('warns on an HTML (rate-limit) page', async () => {
    routes = [
      {
        match: /./,
        reply: { headers: { 'content-type': 'text/html; charset=utf-8' }, body: '<html>' },
      },
    ];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    expect(warnings.at(-1)).toContain('rate limit hit');
  });

  it('tells an exhausted rate limit apart from a permissions problem', async () => {
    routes = [{ match: /./, reply: { status: 403, headers: { 'x-ratelimit-remaining': '0' } } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    expect(warnings.at(-1)).toContain('rate limit exhausted');
    routes = [{ match: /./, reply: { status: 403 } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    expect(warnings.at(-1)).toContain('access denied (403)');
  });

  it('warns on 429 with and without Retry-After', async () => {
    routes = [{ match: /./, reply: { status: 429, headers: { 'retry-after': '30' } } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    expect(warnings.at(-1)).toContain('retry after 30s');
    routes = [{ match: /./, reply: { status: 429 } }];
    await fetchCategoryIndex('design');
    expect(warnings.at(-1)).toContain('retry after unknown');
  });

  it('returns nothing for other statuses, bad JSON and network errors', async () => {
    routes = [{ match: /./, reply: { status: 500 } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    routes = [{ match: /./, reply: { body: '<<not json' } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
    routes = [{ match: /./, reply: { error: new Error('dns') } }];
    expect(await fetchCategoryIndex('design')).toEqual([]);
  });
});

describe('resolveAgents', () => {
  it('takes the fast path when the slug starts with a category', async () => {
    routes = [
      { match: /engineering\/engineering-backend\.md/, reply: { body: agentMd('Backend') } },
    ];
    const [a] = await resolveAgents(['Engineering Backend']);
    expect(a.slug).toBe('backend');
    expect(calls).toHaveLength(1); // no index lookups needed
  });

  it('falls back to scanning category indexes, matching with or without .md', async () => {
    routes = [
      {
        match: /contents\/design/,
        reply: { body: JSON.stringify([{ name: 'other.md', type: 'file' }]) },
      },
      {
        match: /contents\/engineering/,
        reply: { body: JSON.stringify([{ name: 'backend.md', type: 'file' }]) },
      },
      { match: /engineering\/backend\.md/, reply: { body: agentMd('Backend') } },
    ];
    const [a] = await resolveAgents(['backend']);
    expect(a.category).toBe('engineering');
    const order = calls.map((c) => c.url).filter((u) => u.includes('/contents/'));
    expect(order[0]).toContain('/contents/design');
  });

  it('keeps scanning when a matching file cannot be parsed', async () => {
    routes = [
      {
        match: /contents\/design/,
        reply: { body: JSON.stringify([{ name: 'x.md', type: 'file' }]) },
      },
      { match: /design\/x\.md/, reply: { body: 'broken' } },
      {
        match: /contents\/engineering/,
        reply: { body: JSON.stringify([{ name: 'x.md', type: 'file' }]) },
      },
      { match: /engineering\/x\.md/, reply: { body: agentMd('X') } },
    ];
    const [a] = await resolveAgents(['x']);
    expect(a.category).toBe('engineering');
  });

  it('falls through from a failed fast path to the scan', async () => {
    routes = [
      // design/design-x.md does not exist (404), so the category indexes are scanned
      {
        match: /contents\/engineering/,
        reply: { body: JSON.stringify([{ name: 'design-x.md', type: 'file' }]) },
      },
      { match: /engineering\/design-x\.md/, reply: { body: agentMd('DX') } },
    ];
    const [a] = await resolveAgents(['design-x']);
    expect(a.name).toBe('DX');
    expect(a.category).toBe('engineering');
  });

  it('resolves several agents in order', async () => {
    routes = [
      { match: /design-a\.md/, reply: { body: agentMd('A') } },
      { match: /testing-b\.md/, reply: { body: agentMd('B') } },
    ];
    const out = await resolveAgents(['design-a', 'testing-b']);
    expect(out.map((x) => x.name)).toEqual(['A', 'B']);
  });

  it('fails with a helpful message when nothing matches', async () => {
    await expect(resolveAgents(['ghost'])).rejects.toThrow(
      /Agent "ghost" not found in el-j\/agency-agents@main/,
    );
    expect(calls.filter((c) => c.url.includes('/contents/'))).toHaveLength(AGENT_CATEGORIES.length);
  });
});

describe('resolveCategory', () => {
  it('rejects an unknown category without any request', async () => {
    await expect(resolveCategory('nope')).rejects.toThrow('Unknown category "nope"');
    expect(calls).toHaveLength(0);
  });

  it('returns every parsable agent of a category', async () => {
    routes = [
      {
        match: /contents\/design/,
        reply: {
          body: JSON.stringify([
            { name: 'a.md', type: 'file' },
            { name: 'bad.md', type: 'file' },
          ]),
        },
      },
      { match: /design\/a\.md/, reply: { body: agentMd('A') } },
      { match: /design\/bad\.md/, reply: { body: 'broken' } },
    ];
    const out = await resolveCategory('design');
    expect(out.map((a) => a.name)).toEqual(['A']);
  });
});
