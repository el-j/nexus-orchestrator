export interface DaemonOptions {
    readonly binPath: string;
    readonly listenAddr: string;
    readonly mcpAddr: string;
    readonly dbPath: string;
    readonly openaiApiKey: string;
    readonly openaiModel: string;
    readonly anthropicApiKey: string;
    readonly anthropicModel: string;
    readonly githubCopilotToken: string;
    readonly githubCopilotModel: string;
    readonly logFile: string;
}
export interface DaemonHandle {
    readonly pid: number;
    readonly logFile: string;
    stop(): void;
}
/** Spawn nexus-daemon and wait for its health endpoint to respond */
export declare function startDaemon(opts: DaemonOptions): Promise<DaemonHandle>;
/** Poll the health URL until it responds 200 or timeout elapses */
export declare function waitForHealth(url: string, timeoutMs: number): Promise<void>;
/** Print daemon log to GitHub Actions log group */
export declare function printDaemonLog(logFile: string): void;
