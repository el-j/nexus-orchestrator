// Starts what the real-daemon Playwright project talks to: a fake Ollama server
// and the built nexus-daemon, on the ports the Vite dev server proxies to
// (63987 REST, 63988 MCP). Stays in the foreground until Playwright stops it.
import { spawn } from 'node:child_process';
import { createServer } from 'node:http';
import { mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';

const daemonBin = resolve(process.env.NEXUS_DAEMON_BIN ?? '../build/bin/nexus-daemon-e2e');
const work = mkdtempSync(join(tmpdir(), 'nexus-pw-'));

function reply(prompt) {
  const name = /FUNC_([A-Za-z]+)/.exec(prompt)?.[1] ?? 'Generated';
  return '```go\npackage main\n\nfunc ' + name + '() {}\n```';
}

const llm = createServer((req, res) => {
  let body = '';
  req.on('data', (c) => (body += c));
  req.on('end', () => {
    res.setHeader('content-type', 'application/json');
    if (req.url === '/api/tags') return res.end('{"models":[{"name":"fake-coder"}]}');
    if (req.url === '/api/show') return res.end('{"model_info":{"llama.context_length":32768}}');
    const parsed = body ? JSON.parse(body) : {};
    if (req.url === '/api/generate')
      return res.end(JSON.stringify({ response: reply(parsed.prompt ?? '') }));
    if (req.url === '/api/chat') {
      const last = parsed.messages?.at(-1)?.content ?? '';
      return res.end(JSON.stringify({ message: { content: reply(last) } }));
    }
    res.statusCode = 404;
    res.end('{}');
  });
});

await new Promise((ok) => llm.listen(0, '127.0.0.1', ok));
const llmURL = `http://127.0.0.1:${llm.address().port}`;

const daemon = spawn(daemonBin, [], {
  stdio: 'inherit',
  env: {
    ...process.env,
    HOME: work,
    USERPROFILE: work,
    NEXUS_DB_PATH: join(work, 'nexus.db'),
    NEXUS_LISTEN_ADDR: '127.0.0.1:63987',
    NEXUS_MCP_ADDR: '127.0.0.1:63988',
    NEXUS_OLLAMA_URL: llmURL,
    NEXUS_LMSTUDIO_URL: 'http://127.0.0.1:1',
    NEXUS_SCAN_INTERVAL: '1h',
  },
});

function shutdown() {
  daemon.kill('SIGTERM');
  llm.close();
  rmSync(work, { recursive: true, force: true });
}
process.on('SIGTERM', () => {
  shutdown();
  process.exit(0);
});
process.on('SIGINT', () => {
  shutdown();
  process.exit(0);
});
daemon.on('exit', (code) => {
  llm.close();
  process.exit(code ?? 1);
});
