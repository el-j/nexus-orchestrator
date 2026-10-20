/** All inputs the action accepts. Mirrors action.yml inputs. */
export interface ActionInputs {
  // Task content
  readonly instruction: string;
  readonly taskFile: string;
  readonly projectPath: string;
  readonly targetFile: string;
  readonly contextFiles: string[];
  readonly command: string; // validated to 'execute' | 'plan' | 'auto' at startup
  // Routing
  readonly model: string;
  readonly provider: string;
  // Agent identity (agency-agents integration)
  readonly agent: string;
  readonly agents: string;
  readonly agentCategory: string;
  readonly agentRef: string; // git ref for el-j/agency-agents (default: "main")
  readonly systemPrompt: string; // manual system prompt override
  // Daemon
  readonly daemonUrl: string;
  /** Bearer token for a daemon that runs with an API token (`NEXUS_API_TOKEN`); empty for none. */
  readonly apiToken: string;
  readonly startDaemon: boolean;
  readonly nexusVersion: string;
  readonly timeoutSeconds: number;
  // Provider credentials
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
  /** For a WAITING task: when the providers it waits for are expected back (ISO 8601). */
  readonly waitUntil?: string;
}

/** Task lifecycle status as reported by the daemon. */
export type TaskStatus =
  | 'QUEUED'
  | 'WAITING'
  | 'PROCESSING'
  | 'COMPLETED'
  | 'FAILED'
  | 'CANCELLED'
  | 'TOO_LARGE'
  | 'NO_PROVIDER';

/** Statuses after which a task never changes again. */
export const TERMINAL_STATUSES = new Set<TaskStatus>([
  'COMPLETED',
  'FAILED',
  'CANCELLED',
  'TOO_LARGE',
]);

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
