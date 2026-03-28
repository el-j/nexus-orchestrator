import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import HistoryView from './HistoryView.vue';
import { setProject } from '../composables/useProjectState';

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl,
}));

describe('HistoryView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveServerUrl.mockResolvedValue('http://daemon');
    setProject('/repo/a');
  });

  it('requests history scoped to the current project', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        {
          id: 'task-a',
          projectPath: '/repo/a',
          targetFile: 'src/main.go',
          instruction: 'Build feature A',
          contextFiles: [],
          modelId: '',
          providerHint: '',
          command: 'auto',
          status: 'COMPLETED',
          createdAt: '2026-03-28T10:00:00.000Z',
          updatedAt: '2026-03-28T10:05:00.000Z',
          logs: '',
        },
      ],
    });
    vi.stubGlobal('fetch', fetchMock);

    const wrapper = mount(HistoryView, {
      global: {
        stubs: {
          TaskStatusBadge: {
            template: '<div class="status-badge"><slot /></div>',
            props: ['status'],
          },
          TaskDetailDrawer: {
            template: '<div class="detail-drawer" />',
            props: ['modelValue', 'task'],
          },
        },
      },
    });

    await flushPromises();

    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/tasks/all?projectPath=%2Frepo%2Fa');
    expect(wrapper.text()).toContain('task-a');
    expect(wrapper.text()).toContain('a');

    wrapper.unmount();
    setProject(null);
  });
});
