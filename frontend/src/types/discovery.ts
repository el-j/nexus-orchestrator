export type DiscoveryMethod = 'port' | 'cli' | 'process';
export type DiscoveryStatus = 'reachable' | 'installed' | 'running';

// Canonical DiscoveredProvider lives in domain.ts — re-export for backward compat
export type { DiscoveredProvider } from './domain';

export interface LogEntry {
  timestamp: string; // ISO
  level: 'info' | 'warn' | 'error' | 'debug';
  source: string; // e.g. "httpapi", "orchestrator", "lmstudio"
  message: string;
}
