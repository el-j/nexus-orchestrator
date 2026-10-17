import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ProviderLimitBadges from './ProviderLimitBadges.vue';
import type { ProviderInfo } from '../types/domain';

const base: ProviderInfo = { name: 'Claude', active: true, activeModel: 'haiku', models: [] };
const badges = (p: Partial<ProviderInfo>) =>
  mount(ProviderLimitBadges, { props: { provider: { ...base, ...p } } });

describe('ProviderLimitBadges', () => {
  it('shows nothing for a provider with no limit and no quota reading', () => {
    const w = badges({});
    expect(w.find('[data-testid=cooling-badge]').exists()).toBe(false);
    expect(w.find('[data-testid=quota-badge]').exists()).toBe(false);
  });

  it('shows the cooldown with the time and the reason', () => {
    const w = badges({
      rateLimited: true,
      rateLimitedUntil: '2026-10-17T09:30:00Z',
      cooldownReason: 'seven_day limit reached',
    });
    const b = w.get('[data-testid=cooling-badge]');
    expect(b.text()).toMatch(/^cooling down until \d{1,2}:\d{2}/);
    expect(b.attributes('title')).toBe('seven_day limit reached');
  });

  it('shows a cooldown without a time when it is missing or unreadable', () => {
    expect(badges({ rateLimited: true }).get('[data-testid=cooling-badge]').text()).toBe(
      'cooling down',
    );
    expect(
      badges({ rateLimited: true, rateLimitedUntil: 'not a date' })
        .get('[data-testid=cooling-badge]')
        .text(),
    ).toBe('cooling down');
  });

  it('shows the share of the plan used, red from 90 percent', () => {
    const warn = badges({
      quota: { window: 'seven_day', status: 'allowed_warning', utilization: 0.93 },
    });
    expect(warn.get('[data-testid=quota-badge]').text()).toBe('plan 93% used');
    expect(warn.get('[data-testid=quota-badge]').classes()).toContain('bg-red-500/20');
    expect(warn.get('[data-testid=quota-badge]').attributes('title')).toBe('seven_day window');
    const calm = badges({ quota: { window: 'five_hour', status: 'allowed', utilization: 0.4 } });
    expect(calm.get('[data-testid=quota-badge]').classes()).toContain('bg-slate-500/20');
  });

  it('shows no share when the client did not report one', () => {
    expect(
      badges({ quota: { window: 'five_hour', status: 'allowed', utilization: -1 } })
        .find('[data-testid=quota-badge]')
        .exists(),
    ).toBe(false);
  });
});
