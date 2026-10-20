import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import RoutingPreviewCard from './RoutingPreviewCard.vue';

const h = vi.hoisted(() => ({
  getRouting: vi.fn(),
  project: { value: null as string | null },
}));

vi.mock('../types/wails', () => ({ getRouting: h.getRouting }));
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});

describe('RoutingPreviewCard', () => {
  beforeEach(() => {
    h.getRouting.mockReset();
    h.project.value = null;
  });

  it('loads routing preview for the selected project and role, displaying ladder, order, cooldowns and quota', async () => {
    h.project.value = '/work/app';
    h.getRouting.mockResolvedValue({
      role: 'coder',
      ladder: ['Ollama/*', 'Anthropic/claude-sonnet*'],
      quotaThreshold: 0.9,
      order: [
        {
          position: 1,
          provider: 'Ollama',
          model: 'qwen',
          rung: 0,
          utilization: 0.45,
        },
        {
          position: 2,
          provider: 'Anthropic',
          model: 'claude-sonnet',
          rung: 1,
          coolingUntil: '2026-06-01T12:05:00Z',
          cooldownReason: 'rate limited',
          utilization: 0.95,
        },
      ],
    });

    const w = mount(RoutingPreviewCard);
    await flushPromises();

    expect(h.getRouting).toHaveBeenCalledWith('/work/app', '');
    expect(w.text()).toContain('Routing Preview');
    expect(w.text()).toContain('Ollama/*');
    expect(w.text()).toContain('Anthropic/claude-sonnet*');
    expect(w.text()).toContain('90%');
    expect(w.text()).toContain('45%');
    expect(w.text()).toContain('95%');
    expect(w.text()).toContain('rate limited');

    await w.find('[data-testid="routing-role-input"]').setValue('reviewer');
    await w.find('[data-testid="routing-preview-btn"]').trigger('click');
    await flushPromises();
    expect(h.getRouting).toHaveBeenCalledWith('/work/app', 'reviewer');

    await w.find('[data-testid="routing-project-input"]').setValue('/work/typed');
    await w.find('[data-testid="routing-preview-btn"]').trigger('click');
    await flushPromises();
    expect(h.getRouting).toHaveBeenLastCalledWith('/work/typed', 'reviewer');
    w.unmount(); // a mounted card keeps watching the shared project and would answer the next test's call
  });

  it('displays an error message when getRouting rejects and reloads when currentProject changes', async () => {
    h.getRouting.mockRejectedValueOnce(new Error('no providers available'));
    const w = mount(RoutingPreviewCard);
    await flushPromises();
    expect(w.text()).toContain('no providers available');

    h.getRouting.mockRejectedValueOnce('weird');
    await w.find('[data-testid="routing-preview-btn"]').trigger('click');
    await flushPromises();
    expect(w.text()).toContain('Failed to load routing preview');

    h.getRouting.mockResolvedValueOnce({
      quotaThreshold: 0.85,
      order: [{ position: 1, provider: 'LM Studio', rung: -1 }],
    });
    h.project.value = '/work/other';
    await nextTick();
    await flushPromises();
    expect(h.getRouting).toHaveBeenCalledWith('/work/other', '');
    expect(w.text()).toContain('LM Studio');
    expect(w.text()).toContain('unlisted');
  });
});
