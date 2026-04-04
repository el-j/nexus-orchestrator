import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import AgentsView from './AgentsView.vue';

// Mock wails bindings
vi.mock('../types/wails', () => ({
  listAISessions: vi.fn().mockResolvedValue([
    {
      id: 's1',
      agentName: 'copilot-vscode',
      status: 'active',
      source: 'mcp',
      lastActivity: new Date().toISOString(),
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
  ]),
  deregisterAISession: vi.fn(),
  purgeDisconnectedSessions: vi.fn(),
}));

// Mock useServerUrl
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://localhost:63987'),
}));

// Mock fetch for discovered agents + activities
const mockFetch = vi.fn().mockImplementation((url: string) => {
  if (url.includes('/api/ai-sessions/discovered')) {
    return Promise.resolve({
      ok: true,
      json: () =>
        Promise.resolve([
          {
            id: 'a1',
            kind: 'claude-cli',
            name: 'Claude CLI test',
            isRunning: true,
            detectionMethod: 'process',
            lastSeen: new Date().toISOString(),
          },
        ]),
    });
  }
  if (url.includes('/api/activities')) {
    return Promise.resolve({
      ok: true,
      json: () => Promise.resolve([]),
    });
  }
  return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
});
vi.stubGlobal('fetch', mockFetch);

describe('AgentsView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.stubGlobal('fetch', mockFetch);
  });

  it('renders without errors', () => {
    const wrapper = shallowMount(AgentsView);
    expect(wrapper.exists()).toBe(true);
  });

  it('shows the header with "Agents" title', () => {
    const wrapper = shallowMount(AgentsView);
    expect(wrapper.find('header').text()).toContain('Agents');
  });

  it('renders Sessions and Activity tabs', () => {
    const wrapper = shallowMount(AgentsView);
    const html = wrapper.html();
    expect(html).toContain('Sessions');
    expect(html).toContain('Activity');
  });

  it('displays session cards after data loads', async () => {
    const wrapper = shallowMount(AgentsView);
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('copilot-vscode');
    expect(html).toContain('Registered Sessions');
  });

  it('loads discovered agents via fetch', async () => {
    const wrapper = shallowMount(AgentsView);
    await flushPromises();

    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining('/api/ai-sessions/discovered'));
  });

  it('shows discovered agents section when agents exist', async () => {
    const wrapper = shallowMount(AgentsView);
    await flushPromises();

    const html = wrapper.html();
    expect(html).toContain('Discovered Agents');
    expect(html).toContain('Claude CLI test');
  });

  it('has a Refresh button', () => {
    const wrapper = shallowMount(AgentsView);
    const refreshBtn = wrapper.findAll('button').find((b) => b.text().includes('Refresh'));
    expect(refreshBtn).toBeDefined();
  });
});
