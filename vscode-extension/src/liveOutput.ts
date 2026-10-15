/**
 * liveOutput.ts: shows the text of running tasks in an output channel as the model writes it.
 *
 * Follows the daemon's event stream (`task.chunk`, `task.restart`) and prints each task's text
 * under a short `[task-id]` header. An output channel cannot erase text, so a restart (retry, next
 * provider, next agent turn) is marked with a line instead. Reconnects while the daemon is away.
 */
import * as vscode from 'vscode';
import type { NexusClient } from './nexusClient';

/** Pause before reconnecting to the event stream, in milliseconds. */
const RECONNECT_MS = 5_000;

/** Task events that end a task's text; printed as a closing line. */
const ENDINGS: Record<string, string> = {
  'task.completed': 'completed',
  'task.failed': 'failed',
  'task.cancelled': 'cancelled',
  'task.too_large': 'too large',
  'task.no_provider': 'no provider',
};

interface StreamEvent {
  type?: string;
  taskId?: string;
  text?: string;
}

/** Streams running tasks' output into a VS Code output channel. */
export class LiveOutput implements vscode.Disposable {
  private readonly channel = vscode.window.createOutputChannel('Nexus Live Output');
  private readonly abort = new AbortController();
  private readonly shown = new Set<string>();
  private current = '';
  private disposed = false;

  constructor(private readonly client: NexusClient) {}

  /** Registers the command, starts following the stream when enabled and cleans up on disposal. */
  register(context: vscode.ExtensionContext): void {
    context.subscriptions.push(
      this,
      vscode.commands.registerCommand('nexus.showLiveOutput', () => this.channel.show(true)),
    );
    const enabled = vscode.workspace
      .getConfiguration('nexus.liveOutput')
      .get<boolean>('enabled', true);
    if (enabled) void this.follow();
  }

  /** Reads the event stream until disposed, reconnecting after a failure or a closed stream. */
  async follow(): Promise<void> {
    while (!this.disposed) {
      try {
        await this.read(await this.client.openEvents(this.abort.signal));
      } catch {
        /* daemon away or stream broken: wait and retry */
      }
      if (this.disposed) return;
      await new Promise((resolve) => setTimeout(resolve, RECONNECT_MS));
    }
  }

  private async read(resp: Response): Promise<void> {
    const decoder = new TextDecoder();
    let buffer = '';
    for await (const part of resp.body as unknown as AsyncIterable<Uint8Array>) {
      buffer += decoder.decode(part, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() as string; // split always yields one element
      for (const line of lines) {
        if (!line.startsWith('data: ')) continue;
        try {
          this.show(JSON.parse(line.slice(6)) as StreamEvent);
        } catch {
          /* a malformed frame is skipped */
        }
      }
    }
  }

  private show(ev: StreamEvent): void {
    if (!ev.taskId || !ev.type) return;
    const tag = `[${ev.taskId.slice(0, 8)}]`;
    if (ev.type === 'task.restart') {
      this.current = '';
      if (this.shown.has(ev.taskId)) this.channel.append(`\n${tag} ↻ new attempt\n`);
    } else if (ev.type === 'task.chunk' && ev.text) {
      if (this.current !== ev.taskId) this.channel.append(`\n${tag} `);
      this.current = ev.taskId;
      this.shown.add(ev.taskId);
      this.channel.append(ev.text);
    } else if (ev.type in ENDINGS && this.shown.delete(ev.taskId)) {
      this.current = '';
      this.channel.appendLine(`\n${tag} ${ENDINGS[ev.type]}`);
    }
  }

  /** Stops following and closes the channel. */
  dispose(): void {
    this.disposed = true;
    this.abort.abort();
    this.channel.dispose();
  }
}
