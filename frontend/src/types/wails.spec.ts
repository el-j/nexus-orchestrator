import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import * as api from './wails';

type Json = unknown;
const reply = (status: number, body?: Json) =>
  Promise.resolve(
    new Response(body === undefined || status === 204 ? null : JSON.stringify(body), {
      status,
      headers: { 'Content-Type': 'application/json' },
    }),
  );

const fetchMock = vi.fn();
beforeEach(() => {
  fetchMock.mockReset();
  vi.stubGlobal('fetch', fetchMock);
  delete (window as { go?: unknown }).go;
});
afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

/** Installs fake Wails bindings that resolve to `value`, returning the spy for the named method. */
function wails(method: string, value: Json = undefined) {
  const spy = vi.fn().mockResolvedValue(value);
  (window as { go?: unknown }).go = { main: { App: { [method]: spy } } };
  return spy;
}
const lastCall = () => {
  const calls = fetchMock.mock.calls;
  const [path, init] = calls[calls.length - 1];
  return {
    path: path as string,
    method: (init?.method ?? 'GET') as string,
    body: init?.body ? JSON.parse(init.body) : undefined,
    init,
  };
};

/**
 * Every API function: how it is called, which binding it uses in the desktop app,
 * and which REST request it makes in the browser.
 */
interface Case {
  name: keyof typeof api;
  args: unknown[];
  binding: string;
  http: { method: string; path: string; body?: unknown; reply?: Json; status?: number };
  result?: unknown; // expected return value of the HTTP branch
}
const cases: Case[] = [
  {
    name: 'getTask',
    args: ['a b'],
    binding: 'GetTask',
    http: { method: 'GET', path: '/api/tasks/a%20b', reply: { id: 'a b' } },
    result: { id: 'a b' },
  },
  {
    name: 'getQueue',
    args: [],
    binding: 'GetQueue',
    http: { method: 'GET', path: '/api/tasks', reply: [{ id: '1' }] },
    result: [{ id: '1' }],
  },
  {
    name: 'getAllTasks',
    args: [],
    binding: 'GetAllTasks',
    http: { method: 'GET', path: '/api/tasks/all', reply: [] },
    result: [],
  },
  {
    name: 'getProviders',
    args: [],
    binding: 'GetProviders',
    http: { method: 'GET', path: '/api/providers', reply: [{ name: 'x' }] },
    result: [{ name: 'x' }],
  },
  {
    name: 'cancelTask',
    args: ['t1'],
    binding: 'CancelTask',
    http: { method: 'DELETE', path: '/api/tasks/t1', status: 204 },
  },
  {
    name: 'addProviderConfig',
    args: [{ name: 'n' }],
    binding: 'AddProviderConfig',
    http: {
      method: 'POST',
      path: '/api/providers/config',
      body: { name: 'n' },
      reply: { id: 'c' },
    },
    result: { id: 'c' },
  },
  {
    name: 'listProviderConfigs',
    args: [],
    binding: 'ListProviderConfigs',
    http: { method: 'GET', path: '/api/providers/config', reply: [] },
    result: [],
  },
  {
    name: 'updateProviderConfig',
    args: [{ id: 'c1', name: 'n' }],
    binding: 'UpdateProviderConfig',
    http: {
      method: 'PUT',
      path: '/api/providers/config/c1',
      body: { id: 'c1', name: 'n' },
      reply: { id: 'c1' },
    },
    result: { id: 'c1' },
  },
  {
    name: 'removeProviderConfig',
    args: ['c1'],
    binding: 'RemoveProviderConfig',
    http: { method: 'DELETE', path: '/api/providers/config/c1', status: 204 },
  },
  {
    name: 'getDiscoveredProviders',
    args: [],
    binding: 'GetDiscoveredProviders',
    http: { method: 'GET', path: '/api/providers/discovered', reply: [] },
    result: [],
  },
  {
    name: 'triggerScan',
    args: [],
    binding: 'TriggerScan',
    http: { method: 'POST', path: '/api/providers/discovered/scan', status: 204 },
  },
  {
    name: 'promoteTask',
    args: ['t1'],
    binding: 'PromoteTask',
    http: { method: 'POST', path: '/api/tasks/t1/promote', status: 204 },
  },
  {
    name: 'updateTask',
    args: ['t1', { priority: 1 }],
    binding: 'UpdateTask',
    http: { method: 'PUT', path: '/api/tasks/t1', body: { priority: 1 }, reply: { id: 't1' } },
    result: { id: 't1' },
  },
  {
    name: 'listAISessions',
    args: [],
    binding: 'ListAISessions',
    http: { method: 'GET', path: '/api/ai-sessions', reply: [] },
    result: [],
  },
  {
    name: 'registerAISession',
    args: [{ agentName: 'a' }],
    binding: 'RegisterAISession',
    http: {
      method: 'POST',
      path: '/api/ai-sessions',
      body: { agentName: 'a' },
      reply: { id: 's' },
    },
    result: { id: 's' },
  },
  {
    name: 'deregisterAISession',
    args: ['s1'],
    binding: 'DeregisterAISession',
    http: { method: 'DELETE', path: '/api/ai-sessions/s1', status: 204 },
  },
  {
    name: 'claimTask',
    args: ['t1', 's1'],
    binding: 'ClaimTask',
    http: {
      method: 'POST',
      path: '/api/tasks/t1/claim',
      body: { sessionId: 's1' },
      reply: { id: 't1' },
    },
    result: { id: 't1' },
  },
  {
    name: 'updateTaskStatus',
    args: ['t1', 's1', 'COMPLETED', 'ok'],
    binding: 'UpdateTaskStatus',
    http: {
      method: 'PUT',
      path: '/api/tasks/t1/status',
      body: { sessionId: 's1', status: 'COMPLETED', logs: 'ok' },
      reply: { id: 't1' },
    },
    result: { id: 't1' },
  },
  {
    name: 'getRuntimeConfig',
    args: [],
    binding: 'GetRuntimeConfig',
    http: { method: 'GET', path: '/api/config', reply: { queueCap: 5 } },
    result: { queueCap: 5 },
  },
  {
    name: 'updateRuntimeConfig',
    args: [{ queueCap: 9 }],
    binding: 'UpdateRuntimeConfig',
    http: { method: 'PUT', path: '/api/config', body: { queueCap: 9 }, reply: { queueCap: 9 } },
    result: { queueCap: 9 },
  },
  {
    name: 'getBrainStatus',
    args: ['/p q'],
    binding: 'GetBrainStatus',
    http: {
      method: 'GET',
      path: '/api/brain/status?projectPath=%2Fp%20q',
      reply: { initialized: true },
    },
    result: { initialized: true },
  },
  {
    name: 'getProjectContext',
    args: ['/p', 800],
    binding: 'GetProjectContext',
    http: {
      method: 'POST',
      path: '/api/brain/context',
      body: { projectPath: '/p', maxTokens: 800 },
      reply: { text: 't' },
    },
    result: { text: 't' },
  },
  {
    name: 'getFocusedContext',
    args: ['/p', 'why', 400],
    binding: 'GetFocusedContext',
    http: {
      method: 'POST',
      path: '/api/brain/focused-context',
      body: { projectPath: '/p', question: 'why', maxTokens: 400 },
      reply: { text: 'f' },
    },
    result: { text: 'f' },
  },
  {
    name: 'initProject',
    args: ['/p'],
    binding: 'InitProject',
    http: {
      method: 'POST',
      path: '/api/brain/init',
      body: { projectPath: '/p', claudeMDPath: '' },
      reply: { initialized: true },
    },
    result: { initialized: true },
  },
  {
    name: 'deleteKnowledge',
    args: ['k 1'],
    binding: 'DeleteKnowledge',
    http: { method: 'DELETE', path: '/api/brain/knowledge/k%201', status: 204 },
  },
];

describe.each(cases)('$name', (c) => {
  it('uses the desktop binding when running inside Wails', async () => {
    const spy = wails(c.binding, { from: 'wails' });
    const fn = api[c.name] as (...a: unknown[]) => Promise<unknown>;
    const out = await fn(...c.args);
    expect(spy).toHaveBeenCalledTimes(1);
    expect(out).toEqual(c.http.reply === undefined ? { from: 'wails' } : { from: 'wails' });
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it('calls the REST API in the browser', async () => {
    fetchMock.mockImplementation(() => reply(c.http.status ?? 200, c.http.reply));
    const fn = api[c.name] as (...a: unknown[]) => Promise<unknown>;
    const out = await fn(...c.args);
    const call = lastCall();
    expect(call.method).toBe(c.http.method);
    expect(call.path).toBe(c.http.path);
    expect(call.body).toEqual(c.http.body);
    if (c.http.body !== undefined)
      expect(call.init.headers).toEqual({ 'Content-Type': 'application/json' });
    if (c.result !== undefined) expect(out).toEqual(c.result);
  });

  it('throws the server’s error message on a failed request', async () => {
    fetchMock.mockImplementation(() => reply(429, { error: 'queue is full' }));
    const fn = api[c.name] as (...a: unknown[]) => Promise<unknown>;
    await expect(fn(...c.args)).rejects.toThrow('queue is full');
  });
});

describe('request failures', () => {
  it('falls back to the status line when the error body is not JSON or has no message', async () => {
    fetchMock.mockImplementationOnce(() =>
      Promise.resolve(new Response('<html>', { status: 502 })),
    );
    await expect(api.getQueue()).rejects.toThrow('HTTP 502');
    fetchMock.mockImplementationOnce(() => reply(500, { error: '' }));
    await expect(api.getQueue()).rejects.toThrow('HTTP 500');
    fetchMock.mockImplementationOnce(() => reply(500, { error: 42 }));
    await expect(api.getQueue()).rejects.toThrow('HTTP 500');
  });

  it('propagates network errors', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'));
    await expect(api.getQueue()).rejects.toThrow('Failed to fetch');
  });
});

describe('submitTask', () => {
  const input = {
    projectPath: '/p',
    targetFile: 'a.go',
    instruction: 'x',
    priority: 1,
    tags: ['t'],
  };

  it('returns the task_id of the REST response (not "id")', async () => {
    fetchMock.mockImplementation(() => reply(201, { task_id: 'T-9', status: 'QUEUED' }));
    expect(await api.submitTask(input)).toBe('T-9');
    expect(lastCall()).toMatchObject({ method: 'POST', path: '/api/tasks', body: input });
  });

  it('rejects instead of returning undefined when the queue is full', async () => {
    fetchMock.mockImplementation(() => reply(429, { error: 'queue full' }));
    await expect(api.submitTask(input)).rejects.toThrow('queue full');
  });

  it('uses the binding inside Wails', async () => {
    const spy = wails('SubmitTask', 'T-1');
    expect(await api.submitTask(input)).toBe('T-1');
    expect(spy).toHaveBeenCalledWith(input);
  });
});

describe('createDraft', () => {
  it('returns the id of the draft', async () => {
    fetchMock.mockImplementation(() => reply(201, { id: 'D-1', status: 'DRAFT' }));
    expect(await api.createDraft({ instruction: 'x' })).toBe('D-1');
  });
});

describe('getBacklog', () => {
  it('filters by project when one is given, and not otherwise', async () => {
    fetchMock.mockImplementation(() => reply(200, []));
    await api.getBacklog('/a b');
    expect(lastCall().path).toBe('/api/tasks/backlog?project=%2Fa%20b');
    await api.getBacklog('');
    expect(lastCall().path).toBe('/api/tasks/backlog');
  });

  it('treats null as empty, in both modes', async () => {
    fetchMock.mockImplementation(() => reply(200, null));
    expect(await api.getBacklog('')).toEqual([]);
    wails('GetBacklog', null);
    expect(await api.getBacklog('/p')).toEqual([]);
  });
});

describe('getAllTasks', () => {
  it('treats a null binding result as empty', async () => {
    wails('GetAllTasks', null);
    expect(await api.getAllTasks()).toEqual([]);
  });
});

describe('heartbeatAISession', () => {
  it('posts the heartbeat in the browser', async () => {
    fetchMock.mockImplementation(() => reply(204));
    await api.heartbeatAISession('s 1');
    expect(lastCall()).toMatchObject({ method: 'POST', path: '/api/ai-sessions/s%201/heartbeat' });
  });

  it('calls the binding in Wails', async () => {
    const spy = wails('HeartbeatAISession');
    await api.heartbeatAISession('s1');
    expect(spy).toHaveBeenCalledWith('s1');
  });

  it('only warns when the heartbeat fails (the daemon may be restarting)', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    fetchMock.mockRejectedValue(new Error('down'));
    await expect(api.heartbeatAISession('s1')).resolves.toBeUndefined();
    (window as { go?: unknown }).go = {
      main: { App: { HeartbeatAISession: vi.fn().mockRejectedValue(new Error('gone')) } },
    };
    await expect(api.heartbeatAISession('s1')).resolves.toBeUndefined();
    expect(warn).toHaveBeenCalledTimes(2);
  });
});

describe('purgeDisconnectedSessions', () => {
  it('returns the number of deleted sessions, 0 when the field is missing', async () => {
    fetchMock.mockImplementationOnce(() => reply(200, { deleted: 3 }));
    expect(await api.purgeDisconnectedSessions()).toBe(3);
    expect(lastCall()).toMatchObject({ method: 'DELETE', path: '/api/ai-sessions' });
    fetchMock.mockImplementationOnce(() => reply(200, {}));
    expect(await api.purgeDisconnectedSessions()).toBe(0);
    wails('PurgeDisconnectedSessions', 7);
    expect(await api.purgeDisconnectedSessions()).toBe(7);
  });
});

describe('ingestKnowledge', () => {
  it('returns the number of ingested sections', async () => {
    fetchMock.mockImplementation(() => reply(200, { ingestedSections: 4 }));
    expect(await api.ingestKnowledge('/p', '/p/CLAUDE.md')).toBe(4);
    expect(lastCall()).toMatchObject({
      method: 'POST',
      path: '/api/brain/ingest',
      body: { projectPath: '/p', filePath: '/p/CLAUDE.md' },
    });
    const spy = wails('IngestKnowledge', 2);
    expect(await api.ingestKnowledge('/p', 'f')).toBe(2);
    expect(spy).toHaveBeenCalledWith('/p', 'f');
  });
});

describe('searchKnowledge', () => {
  it('unwraps the {"results": [...]} envelope', async () => {
    fetchMock.mockImplementation(() => reply(200, { results: [{ heading: 'h' }] }));
    expect(await api.searchKnowledge('/p', 'a b', 3)).toEqual([{ heading: 'h' }]);
    expect(lastCall().path).toBe('/api/brain/search?projectPath=%2Fp&q=a%20b&limit=3');
  });

  it('returns an empty list for a missing or null result set', async () => {
    fetchMock.mockImplementationOnce(() => reply(200, {}));
    expect(await api.searchKnowledge('/p', 'q', 1)).toEqual([]);
    fetchMock.mockImplementationOnce(() => reply(200, { results: null }));
    expect(await api.searchKnowledge('/p', 'q', 1)).toEqual([]);
  });

  it('uses the binding in Wails', async () => {
    const spy = wails('SearchKnowledge', [{ heading: 'w' }]);
    expect(await api.searchKnowledge('/p', 'q', 5)).toEqual([{ heading: 'w' }]);
    expect(spy).toHaveBeenCalledWith('/p', 'q', 5);
  });
});

describe('initProject', () => {
  it('passes the CLAUDE.md path through', async () => {
    fetchMock.mockImplementation(() => reply(200, {}));
    await api.initProject('/p', '/p/CLAUDE.md');
    expect(lastCall().body).toEqual({ projectPath: '/p', claudeMDPath: '/p/CLAUDE.md' });
    const spy = wails('InitProject', {});
    await api.initProject('/p');
    expect(spy).toHaveBeenCalledWith('/p', '');
  });
});

describe('listKnowledge', () => {
  it('adds the kind filter only when given, and treats null as empty', async () => {
    fetchMock.mockImplementation(() => reply(200, null));
    expect(await api.listKnowledge('/p')).toEqual([]);
    expect(lastCall().path).toBe('/api/brain/knowledge?projectPath=%2Fp');
    await api.listKnowledge('/p', 'decision');
    expect(lastCall().path).toBe('/api/brain/knowledge?projectPath=%2Fp&kind=decision');
    const spy = wails('ListKnowledge', []);
    await api.listKnowledge('/p');
    expect(spy).toHaveBeenCalledWith('/p', '');
  });
});

describe('getFileMap', () => {
  it('returns filePaths, adds the focus area only when given, and tolerates null', async () => {
    fetchMock.mockImplementationOnce(() => reply(200, { filePaths: ['a.go'] }));
    expect(await api.getFileMap('/p')).toEqual(['a.go']);
    expect(lastCall().path).toBe('/api/brain/file-map?projectPath=%2Fp');
    fetchMock.mockImplementationOnce(() => reply(200, { filePaths: null }));
    expect(await api.getFileMap('/p', 'internal x')).toEqual([]);
    expect(lastCall().path).toBe('/api/brain/file-map?projectPath=%2Fp&focusArea=internal%20x');
    const spy = wails('GetFileMap', ['w.go']);
    expect(await api.getFileMap('/p')).toEqual(['w.go']);
    expect(spy).toHaveBeenCalledWith('/p', '');
  });
});

describe('getServerAddr', () => {
  it('asks the desktop app for its address', async () => {
    wails('GetServerAddr', 'http://127.0.0.1:1234');
    expect(await api.getServerAddr()).toBe('http://127.0.0.1:1234');
  });

  it('falls back to the default address when an older binary lacks the binding', async () => {
    (window as { go?: unknown }).go = {
      main: { App: { GetServerAddr: vi.fn().mockRejectedValue(new Error('no')) } },
    };
    expect(await api.getServerAddr()).toBe('http://127.0.0.1:63987');
  });

  it('uses the default address in the browser unless VITE_SERVER_URL is set', async () => {
    expect(await api.getServerAddr()).toBe('http://127.0.0.1:63987');
    vi.stubEnv('VITE_SERVER_URL', 'http://example:9');
    expect(await api.getServerAddr()).toBe('http://example:9');
    vi.unstubAllEnvs();
  });
});
