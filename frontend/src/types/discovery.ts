/** How a provider was found: port probe, CLI or process. */
export type DiscoveryMethod = 'port' | 'cli' | 'process';
/** How far a discovered provider got: reachable, installed or running. */
export type DiscoveryStatus = 'reachable' | 'installed' | 'running';

// Canonical DiscoveredProvider lives in domain.ts — re-export for backward compat
export type { DiscoveredProvider } from './domain';

/** One daemon log line. */
export interface LogEntry {
  timestamp: string; // ISO
  level: 'info' | 'warn' | 'error' | 'debug';
  source: string; // e.g. "httpapi", "orchestrator", "lmstudio"
  message: string;
}
