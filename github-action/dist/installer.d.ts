/** Resolve the download URL for a nexus-orchestrator release */
export declare function resolveDownloadUrl(version: string): string;
/** Download, extract, and return the path to the nexus-daemon binary */
export declare function installDaemon(version: string): Promise<string>;
