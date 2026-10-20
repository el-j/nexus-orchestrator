import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { nextTick } from 'vue';
import PolicyCard from './PolicyCard.vue';

const h = vi.hoisted(() => ({
  getPolicy: vi.fn(),
  setPolicy: vi.fn(),
  project: { value: null as string | null },
}));

vi.mock('../types/wails', () => ({ getPolicy: h.getPolicy, setPolicy: h.setPolicy }));
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});

const T = (id: string) => `[data-testid="${id}"]`;
const existing = {
  found: true,
  policy: {
    verify: 'go test ./...',
    rules: ['keep files small'],
    quotaThreshold: 0.8,
    contextBudget: 4000,
    budgets: { Claude: { dailyTokens: 300000 } },
    ladder: { coder: ['Ollama/*', 'Anthropic/claude-sonnet*'] },
  },
};

beforeEach(() => {
  h.getPolicy.mockReset();
  h.setPolicy.mockReset();
  h.project.value = null;
});

describe('PolicyCard', () => {
  it('asks for a project when none is picked', () => {
    const w = mount(PolicyCard);
    expect(w.find(T('policy-no-project')).exists()).toBe(true);
    expect(h.getPolicy).not.toHaveBeenCalled();
    w.unmount();
  });

  it('shows the policy of the project and saves edits while keeping the other fields', async () => {
    h.project.value = '/work/app';
    h.getPolicy.mockResolvedValue(existing);
    h.setPolicy.mockImplementation((_p: string, pol: unknown) =>
      Promise.resolve({ found: true, policy: pol }),
    );
    const w = mount(PolicyCard);
    await flushPromises();

    expect((w.find(T('policy-quota')).element as HTMLInputElement).value).toBe('80');
    expect((w.find(T('policy-context')).element as HTMLInputElement).value).toBe('4000');
    expect(w.findAll(T('policy-budget-row'))).toHaveLength(1);
    expect(w.findAll(T('policy-ladder-row'))).toHaveLength(1);

    await w.find(T('policy-quota')).setValue(70);
    await w.find(T('policy-history')).setValue(2000);
    await w.find(T('policy-context')).setValue(0); // 0 removes the cap
    await w.find(T('policy-add-budget')).trigger('click');
    const budget = w.findAll(T('policy-budget-row'));
    await budget[1].findAll('input')[0].setValue('Gemini');
    await budget[1].findAll('input')[1].setValue(5000);
    await w.find(T('policy-add-ladder')).trigger('click');
    const ladder = w.findAll(T('policy-ladder-row'));
    await ladder[1].findAll('input')[0].setValue('tester');
    await ladder[1].findAll('input')[1].setValue('Ollama/*, ');
    await w.find(T('policy-save')).trigger('click');
    await flushPromises();

    expect(h.setPolicy).toHaveBeenCalledWith('/work/app', {
      verify: 'go test ./...',
      rules: ['keep files small'],
      quotaThreshold: 0.7,
      historyBudget: 2000,
      budgets: { Claude: { dailyTokens: 300000 }, Gemini: { dailyTokens: 5000 } },
      ladder: { coder: ['Ollama/*', 'Anthropic/claude-sonnet*'], tester: ['Ollama/*'] },
    });
    expect(w.find(T('policy-saved')).exists()).toBe(true);
    w.unmount();
  });

  it('removes budgets, ladders and caps that were emptied, and incomplete rows are dropped', async () => {
    h.project.value = '/work/app';
    h.getPolicy.mockResolvedValue(existing);
    h.setPolicy.mockResolvedValue({ found: true, policy: {} });
    const w = mount(PolicyCard);
    await flushPromises();

    await w.find(T('policy-quota')).setValue(0);
    await w.find(T('policy-context')).setValue(0);
    await w.findAll(T('policy-budget-row'))[0].find('button').trigger('click');
    await w.findAll(T('policy-ladder-row'))[0].find('button').trigger('click');
    await w.find(T('policy-add-budget')).trigger('click'); // a blank row is ignored
    await w.find(T('policy-add-ladder')).trigger('click');
    await w.find(T('policy-save')).trigger('click');
    await flushPromises();

    expect(h.setPolicy).toHaveBeenCalledWith('/work/app', {
      verify: 'go test ./...',
      rules: ['keep files small'],
    });
    w.unmount();
  });

  it('says so when the project has no policy file, and reloads when the project changes', async () => {
    h.project.value = '/work/new';
    h.getPolicy.mockResolvedValueOnce({ found: false, policy: {} });
    const w = mount(PolicyCard);
    await flushPromises();
    expect(w.text()).toContain('no policy file yet');

    h.getPolicy.mockResolvedValueOnce(existing);
    h.project.value = '/work/app';
    await nextTick();
    await flushPromises();
    expect(h.getPolicy).toHaveBeenLastCalledWith('/work/app');
    expect(w.text()).not.toContain('no policy file yet');

    h.project.value = null; // leaving the project clears nothing and asks for one
    await nextTick();
    expect(w.find(T('policy-no-project')).exists()).toBe(true);
    w.unmount();
  });

  it('shows load and save failures, with a generic text for a non-error rejection', async () => {
    h.project.value = '/work/app';
    h.getPolicy.mockRejectedValueOnce(new Error('policy: broken file'));
    const w = mount(PolicyCard);
    await flushPromises();
    expect(w.find(T('policy-error')).text()).toContain('broken file');

    h.getPolicy.mockRejectedValueOnce('weird');
    h.project.value = '/work/other';
    await nextTick();
    await flushPromises();
    expect(w.find(T('policy-error')).text()).toContain('Failed to load the policy');

    h.getPolicy.mockResolvedValueOnce(existing);
    h.project.value = '/work/app';
    await nextTick();
    await flushPromises();
    h.setPolicy.mockRejectedValueOnce(new Error('policy: quotaThreshold must be between 0 and 1'));
    await w.find(T('policy-save')).trigger('click');
    await flushPromises();
    expect(w.find(T('policy-error')).text()).toContain('quotaThreshold');
    h.setPolicy.mockRejectedValueOnce(42);
    await w.find(T('policy-save')).trigger('click');
    await flushPromises();
    expect(w.find(T('policy-error')).text()).toContain('Failed to save the policy');
    w.unmount();
  });

  it('does nothing on Save without a project', async () => {
    h.project.value = '/work/app';
    h.getPolicy.mockResolvedValue(existing);
    const w = mount(PolicyCard);
    await flushPromises();
    h.project.value = null;
    await nextTick();
    (w.vm as unknown as { $: { setupState: { save: () => Promise<void> } } }).$.setupState.save();
    await flushPromises();
    expect(h.setPolicy).not.toHaveBeenCalled();
    w.unmount();
  });
});
