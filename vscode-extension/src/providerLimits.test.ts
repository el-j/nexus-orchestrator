import { describe, expect, it } from 'vitest';
import type { Provider } from './nexusClient';
import { clock, coolingNote, quotaNote } from './providerLimits';

const hhmm = /^\d{2}:\d{2}$/;
const p = (o: Partial<Provider> = {}): Provider => ({ name: 'Claude', active: true, ...o });

describe('providerLimits', () => {
  it('formats a time as a 24-hour clock', () => {
    expect(clock('2026-06-01T15:04:00Z')).toMatch(hhmm);
  });

  it('describes a provider that cools down, with the reason when there is one', () => {
    const until = new Date(Date.now() + 3600_000).toISOString();
    expect(coolingNote(p({ rateLimited: true, rateLimitedUntil: until }))).toMatch(
      /^cooling down until \d{2}:\d{2}$/,
    );
    expect(
      coolingNote(
        p({ rateLimited: true, rateLimitedUntil: until, cooldownReason: 'rate limited (429)' }),
      ),
    ).toMatch(/^cooling down until \d{2}:\d{2} \(rate limited \(429\)\)$/);
    expect(coolingNote(p())).toBeUndefined();
    expect(coolingNote(p({ rateLimited: true }))).toBeUndefined(); // no time known: not shown
  });

  it('describes a plan that is nearly used up and nothing else', () => {
    const resets = new Date(Date.now() + 3600_000).toISOString();
    expect(
      quotaNote(p({ quota: { window: 'w', status: 's', utilization: 0.93, resetsAt: resets } })),
    ).toMatch(/^plan 93% used \(resets \d{2}:\d{2}\)$/);
    expect(quotaNote(p({ quota: { window: 'w', status: 's', utilization: 0.95 } }))).toBe(
      'plan 95% used',
    );
    expect(quotaNote(p({ quota: { window: 'w', status: 's', utilization: 0.4 } }))).toBeUndefined();
    expect(quotaNote(p({ quota: { window: 'w', status: 's', utilization: -1 } }))).toBeUndefined();
    expect(quotaNote(p())).toBeUndefined();
  });
});
