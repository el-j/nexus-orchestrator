import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import AISessionsView from './AISessionsView.vue';
import type { AISession } from '../types/domain';

const h = vi.hoisted(() => ({
  refresh: vi.fn(),
  deregister: vi.fn(),
  state: { sessions: [] as unknown[], loading: false, error: null as string | null },
}));
vi.mock('../composables/useAISessions', () => ({
  useAISessions: () => ({
    sessions: ref(h.state.sessions),
    loading: ref(h.state.loading),
    error: ref(h.state.error),
    refresh: h.refresh,
    deregister: h.deregister,
  }),
}));

const session = (o: Partial<AISession> = {}): AISession =>
  ({
    id: 's1',
    agentName: 'Copilot',
    status: 'active',
    source: 'vscode',
    lastActivity: '2026-01-01T00:00:00Z',
    ...o,
  }) as AISession;

beforeEach(() => {
  h.refresh.mockReset();
  h.deregister.mockReset();
  h.state = { sessions: [], loading: false, error: null };
});

describe('AISessionsView', () => {
  it('shows the empty state', () => {
    const w = mount(AISessionsView);
    expect(w.text()).toContain('No AI sessions detected');
    expect(w.text()).toContain('0 active of 0 total');
  });

  it('shows a spinner while loading, and nothing else', () => {
    h.state.loading = true;
    h.state.sessions = [session()];
    const w = mount(AISessionsView);
    expect(w.find('.animate-spin').exists()).toBe(true);
    expect(w.text()).not.toContain('Copilot');
  });

  it('shows the error instead of the list', () => {
    h.state.error = 'daemon unreachable';
    h.state.sessions = [session()];
    const w = mount(AISessionsView);
    expect(w.text()).toContain('daemon unreachable');
    expect(w.text()).not.toContain('Copilot');
  });

  it('counts active sessions', () => {
    h.state.sessions = [
      session(),
      session({ id: 's2', status: 'idle' }),
      session({ id: 's3', status: 'disconnected' }),
    ];
    const w = mount(AISessionsView);
    expect(w.text()).toContain('1');
    expect(w.text()).toContain('active of 3 total');
    expect(w.find('header .text-emerald-400').exists()).toBe(true);
  });

  it('renders each session with its source, external id and project', () => {
    h.state.sessions = [
      session({ source: 'mcp', externalId: 'ext-1', projectPath: '/work/app' }),
      session({ id: 's2', source: 'vscode' }),
      session({ id: 's3', source: 'http' }),
    ];
    const t = mount(AISessionsView).text();
    expect(t).toContain('🤖 MCP');
    expect(t).toContain('🔵 VS Code');
    expect(t).toContain('🌐 HTTP');
    expect(t).toContain('ext-1');
    expect(t).toContain('/work/app');
    expect(t).toContain('Last active:');
  });

  it('colours the card by status', () => {
    h.state.sessions = [
      session(),
      session({ id: 's2', status: 'idle' }),
      session({ id: 's3', status: 'disconnected' }),
    ];
    const cards = mount(AISessionsView).findAll('.rounded-xl.border');
    expect(cards[0].classes().join(' ')).toContain('emerald');
    expect(cards[1].classes().join(' ')).toContain('yellow');
    expect(cards[2].classes().join(' ')).toContain('slate');
  });

  it('offers Disconnect for live sessions only, and calls deregister', async () => {
    h.state.sessions = [session(), session({ id: 's2', status: 'disconnected' })];
    const w = mount(AISessionsView);
    const buttons = w.findAll('button').filter((b) => b.text() === 'Disconnect');
    expect(buttons).toHaveLength(1);
    await buttons[0].trigger('click');
    expect(h.deregister).toHaveBeenCalledWith('s1');
  });

  it('refreshes from the header button', async () => {
    const w = mount(AISessionsView);
    await w.find('header button').trigger('click');
    expect(h.refresh).toHaveBeenCalledTimes(1);
  });
});
