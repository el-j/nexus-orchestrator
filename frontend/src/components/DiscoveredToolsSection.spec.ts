import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import DiscoveredToolsSection from './DiscoveredToolsSection.vue';
import type { DiscoveredAgent } from '../types/domain';

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://localhost:63987'),
}));

const agent = (over: Partial<DiscoveredAgent>): DiscoveredAgent =>
  ({
    id: 'x',
    kind: 'copilot',
    name: 'Copilot',
    isRunning: false,
    detectionMethod: 'vscode-extension',
    ...over,
  }) as DiscoveredAgent;

function serve(body: unknown, ok = true) {
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok, json: () => Promise.resolve(body) }));
}

async function mountWith(list: DiscoveredAgent[] | null, sortMode: 'active' | 'name' = 'active') {
  serve(list);
  const w = mount(DiscoveredToolsSection, { props: { sortMode } });
  await flushPromises();
  return w;
}

describe('DiscoveredToolsSection', () => {
  beforeEach(() => vi.restoreAllMocks());

  it('explains how to get a tool when none are detected', async () => {
    const w = await mountWith(null);
    expect(w.text()).toContain('No AI coding tools detected');
  });

  it('keeps an empty list when the daemon is unreachable or errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new Error('down')));
    const down = mount(DiscoveredToolsSection, { props: { sortMode: 'active' } });
    await flushPromises();
    expect(down.text()).toContain('No AI coding tools detected');

    serve([], false);
    const bad = mount(DiscoveredToolsSection, { props: { sortMode: 'active' } });
    await flushPromises();
    expect(bad.text()).toContain('No AI coding tools detected');
  });

  it('groups by kind, running groups first, with a status per tool', async () => {
    const w = await mountWith([
      agent({
        id: '1',
        kind: 'continue',
        name: 'Continue',
        isRunning: false,
        detectionMethod: 'config',
      }),
      agent({ id: '2', kind: 'cursor', name: 'Cursor', isRunning: true }),
      agent({
        id: '3',
        kind: 'mystery',
        name: undefined as never,
        detectionMethod: 'config',
        configPath: '/p',
      }),
      agent({ id: '4', kind: '' as never, name: 'Anon', workingDir: '/w' }),
    ]);
    const text = w.text();
    expect(text.indexOf('Cursor')).toBeLessThan(text.indexOf('Continue'));
    expect(text).toContain('Active');
    expect(text).toContain('Installed');
    expect(text).toContain('Detected');
    expect(text).toContain('AI-native code editor');
  });

  it('sorts by name when asked and filters by status and search text', async () => {
    const list = [
      agent({ id: '1', kind: 'copilot', name: 'Zed', isRunning: true }),
      agent({ id: '2', kind: 'copilot', name: 'Alpha', isRunning: false }),
      agent({ id: '3', kind: 'cline', name: 'Beta', isRunning: false, detectionMethod: 'process' }),
    ];
    const w = await mountWith(list, 'name');
    const names = () => w.findAll('.font-medium.text-xs').map((n) => n.text());
    expect(names().slice(0, 2)).toEqual(['Alpha', 'Zed']);

    const [all, running, installed, detected] = w.findAll('button');
    await running.trigger('click');
    expect(w.text()).toContain('Zed');
    expect(w.text()).not.toContain('Alpha');
    await installed.trigger('click');
    expect(w.text()).toContain('Alpha');
    expect(w.text()).not.toContain('Beta');
    await detected.trigger('click');
    expect(w.text()).toContain('Beta');
    await all.trigger('click');
    await w.find('input').setValue('zzz-none');
    expect(w.text()).toContain('No matching tools');
    await w.find('input').setValue('cline');
    expect(w.text()).toContain('Beta');

    const byActive = await mountWith(list, 'active');
    expect(byActive.findAll('.font-medium.text-xs')[0].text()).toBe('Zed');
  });
});
