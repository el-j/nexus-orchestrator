import { ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import LiveActivityView from './LiveActivityView.vue';

const { useActivities, useDiscovery } = vi.hoisted(() => ({
  useActivities: vi.fn(),
  useDiscovery: vi.fn(),
}));

vi.mock('../composables/useActivities', () => ({
  useActivities,
}));

vi.mock('../composables/useDiscovery', () => ({
  useDiscovery,
}));

describe('LiveActivityView', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useDiscovery.mockReturnValue({ scanning: ref(false), scanNow: vi.fn() });
    useActivities.mockReturnValue({
      activities: ref([
        {
          id: 'a1',
          sessionId: 's1',
          agentName: 'Copilot',
          activityType: 'generation',
          summary: 'Build API',
          projectPath: '/repo/a',
          timestamp: '2026-03-28T10:00:00.000Z',
        },
        {
          id: 'a2',
          sessionId: 's2',
          agentName: 'Claude',
          activityType: 'message',
          summary: 'Fix UI',
          projectPath: '/repo/b',
          timestamp: '2026-03-28T10:01:00.000Z',
        },
      ]),
      loading: ref(false),
    });
  });

  it('filters rendered activity groups by selected project', async () => {
    const wrapper = mount(LiveActivityView, {
      global: {
        stubs: {
          AIActivityCard: {
            template: '<div class="activity-card">{{ activity.summary }}</div>',
            props: ['activity'],
          },
        },
      },
    });

    await flushPromises();
    const selects = wrapper.findAll('select');
    await selects[1].setValue('/repo/a');
    await flushPromises();

    expect(wrapper.text()).toContain('Build API');
    expect(wrapper.text()).not.toContain('Fix UI');
  });
});
