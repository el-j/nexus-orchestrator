/**
 * mcpProxy.ts: a stdio-to-HTTP bridge for MCP, bundled with the extension (dist/mcpProxy.js).
 * Agents such as Antigravity start MCP servers as commands that talk JSON-RPC over stdin and
 * stdout; the Nexus daemon speaks it over HTTP. This reads one JSON-RPC message per line, posts it
 * to the daemon and writes the answer back as one line. The extension host's own Node runs it
 * (ELECTRON_RUN_AS_NODE), so nothing needs to be installed.
 */
import * as readline from 'readline';
import type { Readable, Writable } from 'stream';

/** The slice of fetch the proxy uses; tests pass a fake. */
export type FetchLike = (
  url: string,
  init: { method: string; headers: Record<string, string>; body: string },
) => Promise<{ status: number; text(): Promise<string> }>;

/** Everything the proxy talks to. */
export interface ProxyOptions {
  stdin: Readable;
  stdout: Writable;
  stderr: Writable;
  /** The daemon's MCP endpoint, for example http://127.0.0.1:63988/mcp. */
  url: string;
  /** Bearer token, empty for none. */
  token: string;
  fetch: FetchLike;
}

const errorLine = (id: unknown, code: number, message: string): string =>
  JSON.stringify({ jsonrpc: '2.0', id, error: { code, message } }) + '\n';

/** Proxies messages until stdin ends. */
export async function runProxy(o: ProxyOptions): Promise<void> {
  const lines = readline.createInterface({ input: o.stdin, crlfDelay: Infinity });
  for await (const raw of lines) {
    const line = raw.trim();
    if (line === '') continue;
    let id: unknown = null;
    let isRequest = true;
    try {
      const msg = JSON.parse(line) as { id?: unknown };
      id = msg.id ?? null;
      isRequest = msg.id !== undefined;
    } catch {
      o.stdout.write(errorLine(null, -32700, 'parse error'));
      continue;
    }
    try {
      const resp = await o.fetch(o.url, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Accept: 'application/json',
          ...(o.token ? { Authorization: `Bearer ${o.token}` } : {}),
        },
        body: line,
      });
      const text = (await resp.text()).trim();
      if (resp.status >= 400) {
        if (isRequest)
          o.stdout.write(
            errorLine(
              id,
              -32000,
              `nexus daemon answered HTTP ${resp.status}: ${text.slice(0, 200)}`,
            ),
          );
        continue;
      }
      if (text !== '') o.stdout.write(text.replace(/\r?\n/g, ' ') + '\n');
    } catch (err) {
      const reason = err instanceof Error ? err.message : String(err);
      o.stderr.write(`nexus mcp proxy: ${reason}\n`);
      if (isRequest) o.stdout.write(errorLine(id, -32000, `nexus daemon not reachable: ${reason}`));
    }
  }
}
