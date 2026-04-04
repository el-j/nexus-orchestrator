import { describe, it, expect, vi, beforeEach } from 'vitest';
import { shallowMount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import MissionControlView from './MissionControlView.vue';
import * as wailsMocks from '../types/wails';

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

// Mock PrimeVue useToast
vi.mock('primevue/usetoast', () => ({
  useToast: () => ({ add: vi.fn() }),
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

// ── Additional interaction tests (TASK-502) ───────────────────────────────────

describe('MissionControlView — interactions', () => {
  const STUBS = {
    global: {
      stubs: {
        Skeleton: true,
        TaskStatusBadge: true,
        TaskDetailDrawer: true,
        TaskSubmitForm: true,
      },
    },
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('TaskSubmitForm @submitted event triggers refreshTasks (getQueue called again)', async () => {
    const wrapper = shallowMount(MissionControlView, STUBS);
    await flushPromises();

    const callsBefore = vi.mocked(wailsMocks.getQueue).mock.calls.length;

    // Emit the submitted event from the TaskSubmitForm stub
    await wrapper.findComponent({ name: 'TaskSubmitForm' }).trigger('submitted');
    await flushPromises();

    expect(vi.mocked(wailsMocks.getQueue).mock.calls.length).toBeGreaterThan(callsBefore);
  });

  it('clicking the Active work filter button applies the active CSS class', async () => {
    const wrapper = shallowMount(MissionControlView, STUBS);
    await flushPromises();

    // The filter buttons contain 'All', 'Active', 'Drafts/Backlog'
    const filterButtons = wrapper
      .findAll('button')
      .filter((b) => ['All', 'Active', 'Drafts/Backlog'].includes(b.text().trim()));

    const activeBtn = filterButtons.find((b) => b.text().trim() === 'Active');
    expect(activeBtn).toBeDefined();

    await activeBtn!.trigger('click');
    await nextTick();

    // Active filter button should have violet/active class
    expect(activeBtn!.classes().join(' ')).toMatch(/violet|bg-/);
  });

  it('DRAFT task shows Promote button; clicking it calls promoteTask', async () => {
    vi.mocked(wailsMocks.getQueue).mockResolvedValue([
      {
        id: 'D-mc-1',
        instruction: 'Draft instruction',
        status: 'DRAFT',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        projectPath: '/test',
        targetFile: 'test.go',
        logs: '',
      },
    ]);

    const wrapper = shallowMount(MissionControlView, STUBS);
    await flushPromises();

    // Find the Promote button
    const promoteBtn = wrapper.findAll('button').find((b) => b.text().includes('Promote'));
    expect(promoteBtn).toBeDefined();

    await promoteBtn!.trigger('click');
    await flushPromises();

    expect(vi.mocked(wailsMocks.promoteTask)).toHaveBeenCalledWith('D-mc-1');
  });

  it('Show All History toggle button changes visible history section', async () => {
    const wrapper = shallowMount(MissionControlView, STUBS);
    await flushPromises();

    // "Show All History" button should be present
    const toggleBtn = wrapper.findAll('button').find((b) => b.text().includes('Show All History'));
    expect(toggleBtn).toBeDefined();

    await toggleBtn!.trigger('click');
    await nextTick();

    // After click, button text changes to reflect expanded state
    const htmlAfter = wrapper.html();
    expect(htmlAfter).toContain('Show Recent');
  });

  it('Drafts/Backlog filter hides QUEUED tasks (shows empty state)', async () => {
    // Default mock returns a QUEUED task — selecting Drafts filter should hide it
    vi.mocked(wailsMocks.getQueue).mockResolvedValue([
      {
        id: 'T-q',
        instruction: 'Queue task',
        status: 'QUEUED',
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        projectPath: '/test',
        targetFile: 'test.go',
        logs: '',
      },
    ]);

    const wrapper = shallowMount(MissionControlView, STUBS);
    await flushPromises();

    const filterButtons = wrapper
      .findAll('button')
      .filter((b) => ['All', 'Active', 'Drafts/Backlog'].includes(b.text().trim()));

    const draftsBtn = filterButtons.find((b) => b.text().trim() === 'Drafts/Backlog');
    expect(draftsBtn).toBeDefined();

    await draftsBtn!.trigger('click');
    await nextTick();

    // With only a QUEUED task, drafts view shows empty state
    expect(wrapper.html()).toMatch(/No draft\/backlog|No active/i);
  });
});
