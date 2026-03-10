/** All inputs the action accepts. Mirrors action.yml inputs. */
export interface ActionInputs {
    readonly instruction: string;
    readonly taskFile: string;
    readonly projectPath: string;
    readonly targetFile: string;
    readonly contextFiles: string[];
    readonly command: 'execute' | 'plan';
    readonly model: string;
    readonly provider: string;
    readonly agent: string;
    readonly agents: string;
    readonly agentCategory: string;
    readonly agentRef: string;
    readonly systemPrompt: string;
    readonly daemonUrl: string;
    readonly startDaemon: boolean;
    readonly nexusVersion: string;
    readonly timeoutSeconds: number;
    readonly openaiApiKey: string;
    readonly openaiModel: string;
    readonly anthropicApiKey: string;
    readonly anthropicModel: string;
    readonly githubCopilotToken: string;
    readonly githubCopilotModel: string;
}
/** Parsed nexus-daemon task response */
export interface NexusTask {
    readonly id: string;
    readonly status: TaskStatus;
    readonly logs: string;
    readonly projectPath: string;
    readonly targetFile: string;
    readonly instruction: string;
}
export type TaskStatus = 'QUEUED' | 'PROCESSING' | 'COMPLETED' | 'FAILED' | 'CANCELLED' | 'TOO_LARGE';
export declare const TERMINAL_STATUSES: Set<TaskStatus>;
/** A loaded agent identity */
export interface AgentIdentity {
    readonly name: string;
    readonly slug: string;
    readonly description: string;
    readonly color: string;
    readonly category: string;
    readonly systemPrompt: string;
}
/** Submit task request body */
export interface TaskRequest {
    readonly projectPath: string;
    readonly instruction: string;
    readonly targetFile?: string;
    readonly contextFiles?: string[];
    readonly command?: string;
    readonly modelId?: string;
    readonly providerHint?: string;
}
/** Submit task response */
export interface TaskSubmitResponse {
    readonly task_id: string;
    readonly status: TaskStatus;
}
