import { describe, it, expect, vi, beforeEach } from 'vitest';
import { nextTick } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import UsageCard from './UsageCard.vue';
import * as wails from '../types/wails';

vi.mock('../types/wails', () => ({ getUsage: vi.fn() }));

const full = {
  since: '2026-10-17T00:00:00Z',
  totals: {
    provider: '',
    calls: 3,
    inputTokens: 1_530_000,
    outputTokens: 1_250,
    cachedTokens: 40,
    costUSD: 0.75,
  },
  usage: [
    {
      provider: 'Claude (subscription)',
      model: 'claude-haiku-5-5',
      role: 'architect',
      calls: 2,
      inputTokens: 1_500_000,
      outputTokens: 1_200,
    },
    { provider: 'LM Studio', calls: 1, inputTokens: 999, outputTokens: 50 },
  ],
};

describe('UsageCard', () => {
  beforeEach(() => {
    vi.mocked(wails.getUsage).mockReset();
  });

  it('shows totals and a row per provider, model and role with compact token counts', async () => {
    vi.mocked(wails.getUsage).mockResolvedValue(full);
    const w = mount(UsageCard);
    await flushPromises();
    expect(w.get('[data-testid=usage-totals]').text()).toContain('3 calls');
    expect(w.get('[data-testid=usage-totals]').text()).toContain('1.5M in');
    expect(w.get('[data-testid=usage-totals]').text()).toContain('1.3k out');
    expect(w.get('[data-testid=usage-totals]').text()).toContain('$0.7500');
    const rows = w.findAll('tbody tr');
    expect(rows).toHaveLength(2);
    expect(rows[0]!.text()).toContain('Claude (subscription)');
    expect(rows[0]!.text()).toContain('architect');
    expect(rows[1]!.text()).toContain('—'); // no model, no role
    expect(rows[1]!.text()).toContain('999');
  });

  it('leaves the cost out when the providers report none', async () => {
    vi.mocked(wails.getUsage).mockResolvedValue({
      ...full,
      totals: { ...full.totals, costUSD: undefined },
    });
    const w = mount(UsageCard);
    await flushPromises();
    expect(w.get('[data-testid=usage-totals]').text()).not.toContain('$');
  });

  it('says so when nothing was recorded', async () => {
    vi.mocked(wails.getUsage).mockResolvedValue({
      ...full,
      usage: [],
      totals: { ...full.totals, calls: 0 },
    });
    const w = mount(UsageCard);
    await flushPromises();
    expect(w.text()).toContain('No model calls recorded');
    expect(w.find('table').exists()).toBe(false);
  });

  it('shows Loading while the first answer is pending', async () => {
    vi.mocked(wails.getUsage).mockReturnValue(new Promise(() => undefined));
    const w = mount(UsageCard);
    await nextTick();
    expect(w.text()).toContain('Loading');
  });

  it('shows the error when the daemon cannot answer', async () => {
    vi.mocked(wails.getUsage).mockImplementation(() =>
      Promise.reject(new Error('usage ledger not available')),
    );
    const w = mount(UsageCard);
    await flushPromises();
    expect(w.get('[data-testid=usage-error]').text()).toBe('usage ledger not available');
  });

  it('formats small counts as they are', async () => {
    vi.mocked(wails.getUsage).mockResolvedValue({
      ...full,
      totals: { ...full.totals, inputTokens: 7, outputTokens: 12 },
    });
    const w = mount(UsageCard);
    await flushPromises();
    expect(w.get('[data-testid=usage-totals]').text()).toContain('7 in');
  });
});
