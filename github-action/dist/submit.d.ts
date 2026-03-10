import type { NexusTask, TaskRequest, TaskSubmitResponse } from './types.js';
export declare class NexusClient {
    private readonly client;
    private readonly baseUrl;
    constructor(baseUrl: string);
    /** Submit a task and return its ID + initial status */
    submitTask(req: TaskRequest): Promise<TaskSubmitResponse>;
    /** Poll until a terminal status is reached or timeout elapses */
    waitForTask(taskId: string, timeoutMs: number): Promise<NexusTask>;
    /** Fetch a single task by ID */
    getTask(taskId: string): Promise<NexusTask>;
}
