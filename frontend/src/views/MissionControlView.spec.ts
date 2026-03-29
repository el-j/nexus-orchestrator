import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount } from '@vue/test-utils';
import { nextTick } from 'vue';
import MissionControlView from './MissionControlView.vue';

// Mock wails bindings
vi.mock('../types/wails', () => ({
  getQueue: vi.fn().mockResolvedValue([
    {
      id: 'T-1',
      instruction: 'Test task',
      status: 'QUEUED',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      projectPath: '/test',
      targetFile: 'test.go',
      logs: '',
    },
  ]),
  getAllTasks: vi.fn().mockResolvedValue([]),
  getProviders: vi.fn().mockResolvedValue([
    {
      name: 'LM Studio',
      active: true,
      activeModel: 'test-model',
      models: ['test-model'],
      contextLimit: 32768,
      lastChecked: new Date().toISOString(),
    },
  ]),
  listAISessions: vi.fn().mockResolvedValue([
    {
      id: 's1',
      agentName: 'copilot',
      status: 'active',
      lastActivity: new Date().toISOString(),
      source: 'mcp',
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
  ]),
  cancelTask: vi.fn(),
  promoteTask: vi.fn(),
  createDraft: vi.fn(),
  updateTask: vi.fn(),
  deregisterAISession: vi.fn(),
  purgeDisconnectedSessions: vi.fn(),
  submitTask: vi.fn(),
}));

// Mock useServerUrl
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://localhost:63987'),
}));

// Mock useProjectState
vi.mock('../composables/useProjectState', () => ({
  currentProject: { value: null },
}));

describe('MissionControlView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders without errors via shallowMount', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    expect(wrapper.exists()).toBe(true);
  });

  it('renders the status bar header', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const header = wrapper.find('header');
    expect(header.exists()).toBe(true);
    expect(header.text()).toContain('Mission Control');
  });

  it('has the Active Work section', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const sections = wrapper.findAll('section');
    const activeWork = sections.find((s) => s.text().includes('Active Work'));
    expect(activeWork).toBeDefined();
  });

  it('has the Recent Completions (history) section', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const sections = wrapper.findAll('section');
    const history = sections.find((s) => s.text().includes('Recent Completions'));
    expect(history).toBeDefined();
  });

  it('has the Agents panel', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const html = wrapper.html();
    expect(html).toContain('Agents');
  });

  it('has the Providers panel', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const html = wrapper.html();
    expect(html).toContain('Providers');
  });

  it('has the TaskSubmitForm (submit form)', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    expect(wrapper.findComponent({ name: 'TaskSubmitForm' }).exists()).toBe(true);
  });

  it('renders all 5 panels: status bar, task list, agents, providers, submit form', () => {
    const wrapper = shallowMount(MissionControlView, {
      global: {
        stubs: {
          Skeleton: true,
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
          TaskSubmitForm: true,
        },
      },
    });
    const html = wrapper.html();
    // 1. Status bar
    expect(html).toContain('Mission Control');
    // 2. Task list area (Active Work)
    expect(html).toContain('Active Work');
    // 3. Agents panel
    expect(html).toContain('Agents');
    // 4. Providers panel
    expect(html).toContain('Providers');
    // 5. Submit form
    expect(wrapper.findComponent({ name: 'TaskSubmitForm' }).exists()).toBe(true);
  });
});
