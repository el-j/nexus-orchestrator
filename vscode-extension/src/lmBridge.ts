/**
 * lmBridge.ts: a loopback OpenAI-compatible server in front of VS Code's language-model API.
 *
 * The daemon cannot call `vscode.lm` (it lives inside the editor), so the extension serves
 * `GET /v1/models` and `POST /v1/chat/completions` on 127.0.0.1 and the daemon talks to it like to
 * any OpenAI-compatible provider. Whatever models the editor exposes (GitHub Copilot, and any
 * other extension that registers a chat model) become reachable this way, billed to the
 * subscription of the signed-in account. Every request needs the bearer token created at start.
 */
import * as http from 'http';
import { randomBytes, timingSafeEqual } from 'crypto';
import type { AddressInfo } from 'net';
import * as vscode from 'vscode';

/** Largest request body accepted, in bytes. */
const MAX_BODY_BYTES = 8 * 1024 * 1024;

interface ChatMessage {
  role: string;
  content: unknown;
}

/** A running bridge: where to reach it and how to stop it. */
export interface RunningBridge {
  /** Base URL to give the daemon, ends in /v1. */
  baseUrl: string;
  /** Bearer token the daemon must send. */
  token: string;
  /** Stops the server and aborts requests in flight. */
  close(): Promise<void>;
}

/** Error with an HTTP status, used to turn failures into OpenAI-style error bodies. */
class BridgeError extends Error {
  constructor(
    readonly status: number,
    message: string,
  ) {
    super(message);
  }
}

/** Starts the bridge on a random loopback port. */
export async function startLmBridge(): Promise<RunningBridge> {
  const token = randomBytes(24).toString('hex');
  const inflight = new Set<vscode.CancellationTokenSource>();
  const server = http.createServer((req, res) => {
    void handle(req, res, token, inflight);
  });
  await new Promise<void>((resolve, reject) => {
    server.once('error', reject);
    server.listen(0, '127.0.0.1', resolve);
  });
  const port = (server.address() as AddressInfo).port;
  return {
    baseUrl: `http://127.0.0.1:${port}/v1`,
    token,
    close: () =>
      new Promise<void>((resolve) => {
        for (const source of inflight) source.cancel();
        server.close(() => resolve());
        server.closeAllConnections();
      }),
  };
}

async function handle(
  req: http.IncomingMessage,
  res: http.ServerResponse,
  token: string,
  inflight: Set<vscode.CancellationTokenSource>,
): Promise<void> {
  try {
    if (!authorized(req.headers.authorization, token)) throw new BridgeError(401, 'unauthorized');
    if (req.method === 'GET' && req.url === '/v1/models') {
      return send(res, 200, await listModels());
    }
    if (req.method === 'POST' && req.url === '/v1/chat/completions') {
      const whole = await complete(req, res, inflight);
      if (whole !== undefined) send(res, 200, whole); // undefined: it was streamed
      return;
    }
    throw new BridgeError(404, 'not found');
  } catch (err) {
    const failure = toBridgeError(err);
    const body = { error: { message: failure.message, status: failure.status } };
    if (res.headersSent) {
      res.end(`data: ${JSON.stringify(body)}\n\n`); // mid-stream: the status line is gone
    } else {
      send(res, failure.status, body);
    }
  }
}

/** Constant-time check of the Authorization header. */
function authorized(header: string | undefined, token: string): boolean {
  const given = Buffer.from(header ?? '');
  const want = Buffer.from(`Bearer ${token}`);
  return given.length === want.length && timingSafeEqual(given, want);
}

function send(res: http.ServerResponse, status: number, body: unknown): void {
  res.writeHead(status, { 'Content-Type': 'application/json' });
  res.end(JSON.stringify(body));
}

async function listModels(): Promise<unknown> {
  const models = await vscode.lm.selectChatModels({});
  return {
    object: 'list',
    // context_length lets the daemon refuse a prompt the model cannot take instead of failing it.
    data: models.map((m) => ({
      id: m.id,
      object: 'model',
      owned_by: m.vendor,
      ...(m.maxInputTokens > 0 ? { context_length: m.maxInputTokens } : {}),
    })),
  };
}

async function complete(
  req: http.IncomingMessage,
  res: http.ServerResponse,
  inflight: Set<vscode.CancellationTokenSource>,
): Promise<unknown | undefined> {
  const body = await readJson(req);
  const modelId = typeof body.model === 'string' ? body.model : '';
  const messages = Array.isArray(body.messages) ? (body.messages as ChatMessage[]) : [];
  if (!modelId || messages.length === 0) {
    throw new BridgeError(400, 'model and messages are required');
  }
  const [model] = await vscode.lm.selectChatModels({ id: modelId });
  if (!model) throw new BridgeError(404, `model ${modelId} is not available in this editor`);

  const source = new vscode.CancellationTokenSource();
  inflight.add(source);
  res.once('close', () => source.cancel()); // the daemon gave up: stop spending quota
  try {
    const response = await model.sendRequest(toLmMessages(messages), {}, source.token);
    if (body.stream === true) {
      await streamChunks(res, model.id, response.text);
      return undefined;
    }
    let text = '';
    for await (const part of response.text) text += part;
    return {
      object: 'chat.completion',
      model: model.id,
      choices: [{ index: 0, finish_reason: 'stop', message: { role: 'assistant', content: text } }],
    };
  } finally {
    inflight.delete(source);
    source.dispose();
  }
}

/** Writes the model's text as OpenAI `chat.completion.chunk` server-sent events. */
async function streamChunks(
  res: http.ServerResponse,
  modelId: string,
  parts: AsyncIterable<string>,
): Promise<void> {
  res.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' });
  const chunk = (delta: object, finish: string | null): void => {
    const data = { object: 'chat.completion.chunk', model: modelId };
    res.write(
      `data: ${JSON.stringify({ ...data, choices: [{ index: 0, delta, finish_reason: finish }] })}\n\n`,
    );
  };
  chunk({ role: 'assistant' }, null);
  for await (const part of parts) chunk({ content: part }, null);
  chunk({}, 'stop');
  res.end('data: [DONE]\n\n');
}

/**
 * VS Code chat has no system role: system text is sent as a user message ahead of the rest,
 * and assistant turns keep their role.
 */
function toLmMessages(messages: ChatMessage[]): vscode.LanguageModelChatMessage[] {
  return messages.map((m) => {
    const text = typeof m.content === 'string' ? m.content : JSON.stringify(m.content ?? '');
    return m.role === 'assistant'
      ? vscode.LanguageModelChatMessage.Assistant(text)
      : vscode.LanguageModelChatMessage.User(text);
  });
}

async function readJson(req: http.IncomingMessage): Promise<Record<string, unknown>> {
  const chunks: Buffer[] = [];
  let size = 0;
  for await (const chunk of req) {
    size += (chunk as Buffer).length;
    if (size > MAX_BODY_BYTES) throw new BridgeError(413, 'request body too large');
    chunks.push(chunk as Buffer);
  }
  try {
    const parsed: unknown = JSON.parse(Buffer.concat(chunks).toString('utf8'));
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    /* fall through to the 400 below */
  }
  throw new BridgeError(400, 'body must be a JSON object');
}

/** Maps VS Code's LanguageModelError codes to HTTP statuses the daemon understands. */
function toBridgeError(err: unknown): BridgeError {
  if (err instanceof BridgeError) return err;
  const message = err instanceof Error ? err.message : String(err);
  const code = (err as { code?: string } | null)?.code ?? '';
  if (code === 'NoPermissions') return new BridgeError(403, message);
  if (code === 'NotFound') return new BridgeError(404, message);
  if (code === 'Blocked' || /quota|rate/i.test(message)) return new BridgeError(429, message);
  return new BridgeError(502, message);
}
