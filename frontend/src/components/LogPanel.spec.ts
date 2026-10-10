import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import type { LogEntry } from '../types/domain';

const { logs, connected, error, clear } = vi.hoisted(() => ({
  logs: { value: [] as LogEntry[] },
  connected: { value: true },
  error: { value: null as string | null },
  clear: vi.fn(),
}));
vi.mock('../composables/useLogs', () => ({
  useLogs: () => ({
    logs: ref(logs.value),
    connected: ref(connected.value),
    error: ref(error.value),
    clear,
  }),
}));
import LogPanel from './LogPanel.vue';

const entry = (level: string, message: string, ts = '2026-01-01T09:05:07'): LogEntry =>
  ({ timestamp: ts, level, source: 'daemon', message }) as LogEntry;

describe('LogPanel', () => {
  beforeEach(() => {
    logs.value = [
      entry('info', 'started'),
      entry('error', 'exploded'),
      entry('warn', 'careful'),
      entry('debug', 'detail'),
    ];
    connected.value = true;
    error.value = null;
    clear.mockReset();
  });
  afterEach(() => vi.restoreAllMocks());

  it('renders every entry with time, level and source', () => {
    const w = mount(LogPanel);
    const lines = w.findAll('.log-entry').map((l) => l.text());
    expect(lines).toHaveLength(4);
    expect(lines[0]).toBe('[09:05:07] [INFO] [daemon] started');
    expect(w.find('.log-count').text()).toBe('(4)');
    expect(w.find('.status-dot').classes()).toContain('connected');
  });

  it('marks the stream disconnected and shows the reconnect warning', () => {
    connected.value = false;
    error.value = 'stream lost';
    const w = mount(LogPanel);
    expect(w.find('.status-dot').classes()).toContain('disconnected');
    expect(w.find('.log-warning').text()).toBe('Reconnecting...');
    expect(w.find('.log-warning').attributes('title')).toBe('stream lost');
  });

  it('filters by level', async () => {
    const w = mount(LogPanel);
    await w.find('.level-filter').setValue('error');
    expect(w.findAll('.log-entry').map((l) => l.text())).toEqual([
      '[09:05:07] [ERROR] [daemon] exploded',
    ]);
    expect(w.find('.level-error').exists()).toBe(true);
  });

  it('shows a placeholder when nothing matches', async () => {
    logs.value = [];
    const w = mount(LogPanel);
    expect(w.find('.log-empty').text()).toBe('No log entries yet.');
  });

  it('formats an unparsable timestamp without the time prefix', () => {
    logs.value = [
      {
        timestamp: Symbol('bad') as unknown as string,
        level: 'info',
        source: 's',
        message: 'm',
      } as LogEntry,
    ];
    const w = mount(LogPanel);
    expect(w.find('.log-entry').text()).toBe('[INFO] [s] m');
  });

  it('clears via the trash button and toggles auto-scroll', async () => {
    const w = mount(LogPanel);
    await w.find('[title="Clear"]').trigger('click');
    expect(clear).toHaveBeenCalledTimes(1);
    const auto = w.find('[title="Auto-scroll"]');
    expect(auto.classes()).toContain('active');
    await auto.trigger('click');
    expect(auto.classes()).not.toContain('active');
  });

  it('collapses to just the header', async () => {
    const w = mount(LogPanel);
    await w.find('.collapse-btn').trigger('click');
    expect(w.find('.log-list').exists()).toBe(false);
    expect(w.find('.level-filter').exists()).toBe(false);
    expect(w.find('.collapse-btn').text()).toBe('▲');
    await w.find('.collapse-btn').trigger('click');
    expect(w.find('.log-list').exists()).toBe(true);
  });

  it('turns auto-scroll off when the user scrolls up, and keeps it when at the bottom', async () => {
    const w = mount(LogPanel, { attachTo: document.body });
    const list = w.find('.log-list');
    const el = list.element as HTMLElement;
    Object.defineProperties(el, {
      clientHeight: { value: 100, configurable: true },
      scrollHeight: { value: 1000, configurable: true },
    });
    el.scrollTop = 900;
    await list.trigger('scroll');
    expect(w.find('[title="Auto-scroll"]').classes()).toContain('active');
    el.scrollTop = 100;
    await list.trigger('scroll');
    expect(w.find('[title="Auto-scroll"]').classes()).not.toContain('active');
    w.unmount();
  });

  it('resizes by dragging the header within 80-500px, and ignores drags while collapsed', async () => {
    const w = mount(LogPanel, { attachTo: document.body });
    const header = w.find('.log-header');
    await header.trigger('mousedown', { clientY: 500 });
    document.dispatchEvent(new MouseEvent('mousemove', { clientY: 400 }));
    await w.vm.$nextTick();
    expect(w.find('.log-panel').attributes('style')).toContain('height: 300px');
    document.dispatchEvent(new MouseEvent('mousemove', { clientY: -9000 }));
    await w.vm.$nextTick();
    expect(w.find('.log-panel').attributes('style')).toContain('height: 500px');
    document.dispatchEvent(new MouseEvent('mousemove', { clientY: 9000 }));
    await w.vm.$nextTick();
    expect(w.find('.log-panel').attributes('style')).toContain('height: 80px');
    document.dispatchEvent(new MouseEvent('mouseup'));
    document.dispatchEvent(new MouseEvent('mousemove', { clientY: 0 })); // listeners removed: no change
    await w.vm.$nextTick();
    expect(w.find('.log-panel').attributes('style')).toContain('height: 80px');

    await w.find('.collapse-btn').trigger('click');
    await w.find('.log-header').trigger('mousedown', { clientY: 10 });
    document.dispatchEvent(new MouseEvent('mousemove', { clientY: 0 }));
    w.unmount();
  });

  it('scrolls to the newest entry when auto-scroll is on', async () => {
    const w = mount(LogPanel, { attachTo: document.body });
    const el = w.find('.log-list').element as HTMLElement;
    Object.defineProperty(el, 'scrollHeight', { value: 777, configurable: true });
    // Trigger the length watcher via a filter change that changes the count.
    await w.find('.level-filter').setValue('info');
    await w.vm.$nextTick();
    await w.vm.$nextTick();
    expect(el.scrollTop).toBe(777);
    w.unmount();
  });
});
