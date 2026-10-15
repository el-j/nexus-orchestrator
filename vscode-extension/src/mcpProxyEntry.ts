/** Entry point of dist/mcpProxy.js: wires the proxy to the process's streams and environment. */
import { runProxy } from './mcpProxy';

void runProxy({
  stdin: process.stdin,
  stdout: process.stdout,
  stderr: process.stderr,
  url: process.env['NEXUS_MCP_URL'] || 'http://127.0.0.1:63988/mcp',
  token: (process.env['NEXUS_MCP_TOKEN'] ?? '').trim(),
  fetch: (url, init) => fetch(url, init),
});
