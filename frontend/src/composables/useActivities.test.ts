import { defineComponent, h } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useActivities } from './useActivities';

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('./useServerUrl', () => ({
  resolveServerUrl,
}));

describe('useActivities', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveServerUrl.mockResolvedValue('http://daemon');
  });

  it('passes projectPath to the timeline API and keeps client-side filters working', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => [
        {
          id: 'a1',
          agentName: 'Copilot',
          activityType: 'generation',
          summary: 'Generated API handler',
          projectPath: '/repo/a',
          timestamp: '2026-03-28T10:00:00.000Z',
        },
        {
          id: 'a2',
          agentName: 'Claude',
          activityType: 'message',
          summary: 'Other project',
          projectPath: '/repo/b',
          timestamp: '2026-03-28T10:01:00.000Z',
        },
      ],
    });
    vi.stubGlobal('fetch', fetchMock);

    let state: ReturnType<typeof useActivities>;
    const Harness = defineComponent({
      setup() {
        state = useActivities({
          projectFilter: '/repo/a',
          agentFilter: 'Copilot',
          limit: 5,
        });
        return () => h('div');
      },
    });

    const wrapper = mount(Harness);
    await flushPromises();

    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/activities/timeline?');
    expect(String(fetchMock.mock.calls[0][0])).toContain('limit=5');
    expect(String(fetchMock.mock.calls[0][0])).toContain('projectPath=%2Frepo%2Fa');
    expect(state!.activities.value).toHaveLength(2);
    expect(state!.filtered.value).toHaveLength(1);
    expect(state!.filtered.value[0].summary).toBe('Generated API handler');

    wrapper.unmount();
  });
});
