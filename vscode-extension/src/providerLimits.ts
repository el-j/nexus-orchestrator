/**
 * providerLimits.ts — one-line descriptions of a provider's limits (cooldown after a limit, plan nearly
 * used up) shared by the status bar and the provider pickers.
 */

import type { Provider } from './nexusClient';

/** Plans at or above this share of their allowance are reported as nearly used up. */
export const NEARLY_USED = 0.9;

/** A time of day on a 24-hour clock, for example `15:04`. */
export function clock(iso: string): string {
  return new Date(iso).toLocaleTimeString([], {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  });
}

/** `cooling down until 15:04 (reason)` for a provider that hit a limit, or undefined. */
export function coolingNote(p: Provider): string | undefined {
  if (!p.rateLimited || !p.rateLimitedUntil) return undefined;
  return (
    `cooling down until ${clock(p.rateLimitedUntil)}` +
    (p.cooldownReason ? ` (${p.cooldownReason})` : '')
  );
}

/** `plan 93% used (resets 15:04)` for a plan nearly used up, or undefined. */
export function quotaNote(p: Provider): string | undefined {
  if (!p.quota || p.quota.utilization < NEARLY_USED) return undefined;
  return (
    `plan ${Math.round(p.quota.utilization * 100)}% used` +
    (p.quota.resetsAt ? ` (resets ${clock(p.quota.resetsAt)})` : '')
  );
}
