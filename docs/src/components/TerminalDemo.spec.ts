import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import TerminalDemo from './TerminalDemo.vue';

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

const lines = (w: ReturnType<typeof mount>) => w.findAll('[aria-live] > div');

describe('TerminalDemo', () => {
  it('starts empty, then types its lines one by one', async () => {
    const w = mount(TerminalDemo);
    expect(lines(w)).toHaveLength(0);
    await vi.advanceTimersByTimeAsync(800); // first line appears
    expect(lines(w)).toHaveLength(1);
    expect(lines(w)[0].text()).toContain('$ nexus-daemon');
    await vi.advanceTimersByTimeAsync(400 + 200);
    expect(lines(w).length).toBeGreaterThan(2);
  });

  it('shows a cursor only on the line being typed', async () => {
    const w = mount(TerminalDemo);
    await vi.advanceTimersByTimeAsync(1200);
    expect(w.findAll('.terminal-cursor')).toHaveLength(1);
  });

  it('finishes, drops the cursor, then restarts from the top', async () => {
    const w = mount(TerminalDemo);
    await vi.advanceTimersByTimeAsync(800 + 400 + 200 * 12);
    const finished = lines(w).length;
    expect(finished).toBe(12);
    expect(w.find('.terminal-cursor').exists()).toBe(false);
    await vi.advanceTimersByTimeAsync(4000); // loop: cleared and replaying
    expect(lines(w).length).toBeLessThan(finished);
  });

  it('shows the real response shape of POST /api/tasks', async () => {
    const w = mount(TerminalDemo);
    await vi.advanceTimersByTimeAsync(800 + 400 + 200 * 12);
    expect(w.text()).toContain('"task_id"');
    expect(w.text()).not.toContain('"id":"a1b2c3"');
  });

  it('colours lines by type', async () => {
    const w = mount(TerminalDemo);
    await vi.advanceTimersByTimeAsync(800 + 400 + 200 * 12);
    const classes = lines(w).map((l) => l.classes().join(' '));
    for (const c of [
      'text-violet-400',
      'text-emerald-400',
      'text-cyan-400',
      'text-slate-400',
      'h-3',
    ]) {
      expect(
        classes.some((x) => x.includes(c)),
        c,
      ).toBe(true);
    }
  });

  it('stops its timer when unmounted', async () => {
    const w = mount(TerminalDemo);
    w.unmount();
    await vi.advanceTimersByTimeAsync(10_000);
    expect(vi.getTimerCount()).toBe(0);
  });
});
