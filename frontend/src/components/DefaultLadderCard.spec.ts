import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import DefaultLadderCard from './DefaultLadderCard.vue';

const h = vi.hoisted(() => ({
  getRuntimeConfig: vi.fn(),
  updateRuntimeConfig: vi.fn(),
}));

vi.mock('../types/wails', () => ({
  getRuntimeConfig: h.getRuntimeConfig,
  updateRuntimeConfig: h.updateRuntimeConfig,
}));

describe('DefaultLadderCard', () => {
  beforeEach(() => {
    h.getRuntimeConfig.mockReset();
    h.updateRuntimeConfig.mockReset();
  });

  it('loads and saves the global default ladder', async () => {
    h.getRuntimeConfig.mockResolvedValue({
      queueCap: 50,
      apiTokenEnabled: false,
      mcpTokenEnabled: false,
      defaultLadder: { coder: ['Ollama/*', 'Claude (subscription)/*'] },
    });
    h.updateRuntimeConfig.mockImplementation((u: { defaultLadder?: Record<string, string[]> }) =>
      Promise.resolve({
        queueCap: 50,
        apiTokenEnabled: false,
        mcpTokenEnabled: false,
        defaultLadder: u.defaultLadder,
      }),
    );

    const w = mount(DefaultLadderCard, { props: { suggestions: ['Ollama/*'] } });
    await flushPromises();
    expect(w.findAll('[data-testid="default-ladder-row"]')).toHaveLength(1);

    // Add a blank role (dropped on save) and save
    await w.find('[data-testid="default-ladder-add"]').trigger('click');
    await w.find('[data-testid="default-ladder-save"]').trigger('click');
    await flushPromises();

    expect(h.updateRuntimeConfig).toHaveBeenCalledWith({
      defaultLadder: { coder: ['Ollama/*', 'Claude (subscription)/*'] },
    });
    expect(w.find('[data-testid="default-ladder-saved"]').exists()).toBe(true);
    w.unmount();

    // Missing defaultLadder loads cleanly as empty list
    h.getRuntimeConfig.mockResolvedValueOnce({
      queueCap: 50,
      apiTokenEnabled: false,
      mcpTokenEnabled: false,
    });
    const wEmpty = mount(DefaultLadderCard);
    await flushPromises();
    expect(wEmpty.findAll('[data-testid="default-ladder-row"]')).toHaveLength(0);
    wEmpty.unmount();
  });

  it('surfaces load and save errors for both Error and non-Error rejections', async () => {
    h.getRuntimeConfig.mockRejectedValueOnce(new Error('config down'));
    const w = mount(DefaultLadderCard);
    await flushPromises();
    expect(w.find('[data-testid="default-ladder-error"]').text()).toBe('config down');

    h.updateRuntimeConfig.mockRejectedValueOnce(new Error('save failed'));
    await w.find('[data-testid="default-ladder-save"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="default-ladder-error"]').text()).toBe('save failed');

    h.updateRuntimeConfig.mockRejectedValueOnce('raw save err');
    await w.find('[data-testid="default-ladder-save"]').trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="default-ladder-error"]').text()).toBe(
      'Failed to save global ladder',
    );
    w.unmount();

    h.getRuntimeConfig.mockRejectedValueOnce('raw load err');
    const w2 = mount(DefaultLadderCard);
    await flushPromises();
    expect(w2.find('[data-testid="default-ladder-error"]').text()).toBe(
      'Failed to load global ladder',
    );
    w2.unmount();
  });
});
