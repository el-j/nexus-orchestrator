import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import DiscoveredProvidersPanel from './DiscoveredProvidersPanel.vue';
import type { DiscoveredProvider } from '../types/domain';

const prov = (o: Partial<DiscoveredProvider> = {}): DiscoveredProvider => ({
  id: 'p1',
  name: 'LM Studio',
  kind: 'lmstudio',
  method: 'port',
  status: 'reachable',
  lastSeen: new Date().toISOString(),
  ...o,
});

const mountPanel = (
  providers: DiscoveredProvider[],
  o: { loading?: boolean; scanning?: boolean } = {},
) =>
  mount(DiscoveredProvidersPanel, { props: { providers, loading: false, scanning: false, ...o } });

describe('DiscoveredProvidersPanel', () => {
  it('shows the empty state when nothing was found', () => {
    const w = mountPanel([]);
    expect(w.text()).toContain('No AI tools detected');
  });

  it('shows skeletons (not the empty state) while loading', () => {
    const w = mountPanel([], { loading: true });
    expect(w.text()).not.toContain('No AI tools detected');
    expect(w.findAll('.animate-pulse')).toHaveLength(3);
  });

  it('emits scan from the header button and disables it while scanning', async () => {
    const w = mountPanel([]);
    await w.find('button').trigger('click');
    expect(w.emitted('scan')).toHaveLength(1);

    const busy = mountPanel([], { scanning: true });
    const btn = busy.find('button');
    expect(btn.attributes('disabled')).toBeDefined();
    expect(btn.text()).toBe('Scanning…');
    expect(busy.find('.animate-spin').exists()).toBe(true);
  });

  it('renders details of a reachable provider and emits promote', async () => {
    const p = prov({ baseUrl: 'http://127.0.0.1:1234/v1', models: ['a', 'b'] });
    const w = mountPanel([p]);
    expect(w.text()).toContain('LM Studio');
    expect(w.text()).toContain('API Reachable');
    expect(w.text()).toContain('http://127.0.0.1:1234/v1');
    expect(w.text()).toContain('just now');
    expect(w.find('.pi-wifi').exists()).toBe(true);

    const promote = w.findAll('button').find((b) => b.text().includes('Promote to Active'))!;
    await promote.trigger('click');
    expect(w.emitted('promote')![0]).toEqual([p]);
  });

  it('caps the model list at five and counts the rest', () => {
    const w = mountPanel([prov({ models: ['m1', 'm2', 'm3', 'm4', 'm5', 'm6', 'm7'] })]);
    expect(w.text()).toContain('m5');
    expect(w.text()).not.toContain('m6');
    expect(w.text()).toContain('+2 more');
  });

  it('omits the "+N more" chip when five or fewer models exist', () => {
    const w = mountPanel([prov({ models: ['m1', 'm2', 'm3', 'm4', 'm5'] })]);
    expect(w.text()).not.toContain('more');
  });

  it('offers no promote button for an installed CLI and explains why', () => {
    const w = mountPanel([
      prov({ status: 'installed', method: 'cli', cliPath: '/usr/bin/ollama' }),
    ]);
    expect(w.text()).toContain('Installed');
    expect(w.text()).toContain('/usr/bin/ollama');
    expect(w.text()).toContain('CLI found — start the server to activate');
    expect(w.text()).not.toContain('Promote to Active');
    expect(w.find('.pi-terminal').exists()).toBe(true);
  });

  it('flags a bare process as having no API endpoint', () => {
    const w = mountPanel([prov({ status: 'running', method: 'process', processName: 'ollama' })]);
    expect(w.text()).toContain('Running');
    expect(w.text()).toContain('ollama');
    expect(w.text()).toContain('Process detected — no API endpoint found');
    expect(w.text()).not.toContain('Promote to Active');
  });

  it('falls back gracefully for an unknown method and status', () => {
    const odd = prov({
      method: 'mystery' as DiscoveredProvider['method'],
      status: 'weird' as DiscoveredProvider['status'],
    });
    const w = mountPanel([odd]);
    expect(w.find('.pi-question-circle').exists()).toBe(true);
    expect(w.text()).toContain('weird');
    expect(w.text()).toContain('Process detected — no API endpoint found');
  });

  it('uses a distinct accent per status', () => {
    const w = mountPanel([
      prov({ id: 'a', status: 'reachable' }),
      prov({ id: 'b', status: 'installed' }),
      prov({ id: 'c', status: 'running' }),
      prov({ id: 'd', status: 'weird' as DiscoveredProvider['status'] }),
    ]);
    const cards = w.findAll('.rounded-xl.border.p-4');
    const classes = cards.map((c) => c.classes().join(' '));
    expect(new Set(classes).size).toBe(4);
  });
});
