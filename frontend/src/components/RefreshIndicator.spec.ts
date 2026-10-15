import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import RefreshIndicator from './RefreshIndicator.vue';

describe('RefreshIndicator', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-01-01T12:00:00Z'));
  });
  afterEach(() => vi.useRealTimers());

  const ago = (ms: number) => new Date(Date.now() - ms);

  it.each([
    [1_000, 'just now'],
    [30_000, '30s ago'],
    [5 * 60_000, '5m ago'],
    [3 * 3_600_000, '3h ago'],
  ])('labels %i ms as "%s"', (ms, label) => {
    expect(mount(RefreshIndicator, { props: { lastRefreshed: ago(ms) } }).text()).toBe(label);
  });

  it('is blank until something has been refreshed', () => {
    expect(mount(RefreshIndicator, { props: { lastRefreshed: null } }).text()).toBe('');
  });

  it('keeps ticking so the label ages without a new prop', async () => {
    const wrapper = mount(RefreshIndicator, { props: { lastRefreshed: ago(1_000) } });
    expect(wrapper.text()).toBe('just now');
    vi.advanceTimersByTime(65_000);
    await wrapper.vm.$nextTick();
    expect(wrapper.text()).toBe('1m ago');
  });

  it('stops its timer when unmounted', () => {
    const clear = vi.spyOn(globalThis, 'clearInterval');
    mount(RefreshIndicator, { props: { lastRefreshed: null } }).unmount();
    expect(clear).toHaveBeenCalled();
  });
});
