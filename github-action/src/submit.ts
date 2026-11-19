import * as core from '@actions/core';
import { HttpClient } from '@actions/http-client';
import type { NexusTask, TaskRequest, TaskSubmitResponse } from './types.js';
import { TERMINAL_STATUSES } from './types.js';

/** HTTP 201 Created — @actions/http-client v4 does not export this constant */
const HTTP_CREATED = 201;

/** The subset of the daemon HTTP API the Action needs to submit a task and wait for it. */
export class NexusClient {
  private readonly client: HttpClient;
  private readonly baseUrl: string;

  /** `apiToken` is sent as a bearer token to a daemon that runs with `NEXUS_API_TOKEN`. */
  constructor(baseUrl: string, apiToken?: string) {
    this.baseUrl = baseUrl.replace(/\/$/, '');
    this.client = new HttpClient('nexus-orchestrator-action/1.0', [], {
      allowRetries: true,
      maxRetries: 3,
      ...(apiToken ? { headers: { Authorization: `Bearer ${apiToken}` } } : {}),
    });
  }

  /** Submit a task and return its ID + initial status */
  async submitTask(req: TaskRequest): Promise<TaskSubmitResponse> {
    const res = await this.client.postJson<TaskSubmitResponse>(`${this.baseUrl}/api/tasks`, req);
    if (res.statusCode !== HTTP_CREATED) {
      throw new Error(`POST /api/tasks returned HTTP ${res.statusCode ?? '?'}`);
    }
    if (res.result == null) {
      throw new Error('Empty response from POST /api/tasks');
    }
    return res.result;
  }

  /** Poll until a terminal status is reached or timeout elapses */
  async waitForTask(taskId: string, timeoutMs: number): Promise<NexusTask> {
    const deadline = Date.now() + timeoutMs;
    let lastStatus = 'unknown';
    let resumes = '';
    while (Date.now() < deadline) {
      const task = await this.getTask(taskId);
      lastStatus = task.status;
      resumes = task.status === 'WAITING' && task.waitUntil ? task.waitUntil : '';
      const ts = new Date().toISOString().slice(11, 19);
      core.info(
        `  [${ts}] task=${taskId} status=${task.status}` +
          (resumes ? ` (every provider is at a limit: waiting until ${resumes})` : ''),
      );
      if (TERMINAL_STATUSES.has(task.status)) {
        return task;
      }
      await sleep(5_000);
    }
    // NO_PROVIDER and WAITING are not terminal (a provider may appear, a limit resets), so they
    // would otherwise surface only as a bare timeout; the last status says why.
    throw new Error(
      `Task ${taskId} did not complete within ${timeoutMs / 1000}s (last status: ${lastStatus}${resumes ? `, resumes ${resumes}` : ''})`,
    );
  }

  /** Fetch a single task by ID */
  async getTask(taskId: string): Promise<NexusTask> {
    const res = await this.client.getJson<NexusTask>(`${this.baseUrl}/api/tasks/${taskId}`);
    if (res.result == null) {
      throw new Error(`Task ${taskId} not found`);
    }
    return res.result;
  }
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
