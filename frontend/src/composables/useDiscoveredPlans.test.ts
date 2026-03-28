import { defineComponent, h, nextTick, ref } from 'vue';
import { mount, flushPromises } from '@vue/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDiscoveredPlans } from './useDiscoveredPlans';

const { resolveServerUrl } = vi.hoisted(() => ({
  resolveServerUrl: vi.fn(),
}));

vi.mock('./useServerUrl', () => ({
  resolveServerUrl,
}));

describe('useDiscoveredPlans', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveServerUrl.mockResolvedValue('http://daemon');
  });

  it('fetches discovered plans for the current project and reacts to project changes', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: 'plan-a',
            path: '/repo/a/.claude/plans/PLAN-1.md',
            kind: 'markdown',
            format: 'markdown',
            projectPath: '/repo/a',
            lastModified: '2026-03-28T10:00:00.000Z',
            isActive: true,
          },
        ],
      })
      .mockResolvedValueOnce({
        ok: true,
        json: async () => [
          {
            id: 'plan-b',
            path: '/repo/b/.claude/plans/PLAN-2.md',
            kind: 'markdown',
            format: 'markdown',
            projectPath: '/repo/b',
            lastModified: '2026-03-28T11:00:00.000Z',
            isActive: false,
          },
        ],
      });
    vi.stubGlobal('fetch', fetchMock);

    const projectPath = ref<string | null>('/repo/a');
    let state: ReturnType<typeof useDiscoveredPlans>;
    const Harness = defineComponent({
      setup() {
        state = useDiscoveredPlans(projectPath);
        return () => h('div');
      },
    });

    const wrapper = mount(Harness);
    await flushPromises();

    expect(String(fetchMock.mock.calls[0][0])).toContain(
      '/api/plans/discovered?projectPath=%2Frepo%2Fa',
    );
    expect(state!.plans.value).toHaveLength(1);
    expect(state!.plans.value[0].id).toBe('plan-a');

    projectPath.value = '/repo/b';
    await nextTick();
    await flushPromises();

    expect(String(fetchMock.mock.calls[1][0])).toContain(
      '/api/plans/discovered?projectPath=%2Frepo%2Fb',
    );
    expect(state!.plans.value[0].id).toBe('plan-b');

    wrapper.unmount();
  });
});
