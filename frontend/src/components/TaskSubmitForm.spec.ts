import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { ref } from 'vue';
import TaskSubmitForm from './TaskSubmitForm.vue';

const api = vi.hoisted(() => ({
  submitTask: vi.fn(),
  createDraft: vi.fn(),
  toastAdd: vi.fn(),
}));
vi.mock('../types/wails', () => ({ submitTask: api.submitTask, createDraft: api.createDraft }));
vi.mock('primevue/usetoast', () => ({ useToast: () => ({ add: api.toastAdd }) }));
vi.mock('../composables/useProviders', () => ({
  useProviders: () => ({
    providers: ref([
      { name: 'Ollama', active: true, activeModel: '', models: ['llama3', 'qwen'] },
      { name: 'LM Studio', active: true, activeModel: '', models: [] },
    ]),
  }),
}));

// Minimal v-model-capable stand-ins for the PrimeVue inputs.
const textInput = (tag: 'input' | 'textarea') => ({
  props: ['modelValue', 'placeholder'],
  emits: ['update:modelValue'],
  template: `<${tag} :value="modelValue" :placeholder="placeholder" @input="$emit('update:modelValue', $event.target.value)" />`,
});
const SelectStub = {
  props: ['modelValue', 'options', 'placeholder', 'disabled'],
  emits: ['update:modelValue'],
  template: `<select :data-ph="placeholder" :disabled="disabled" :value="modelValue" @change="$emit('update:modelValue', options.find(o => String(o.value) === $event.target.value).value)">
    <option v-for="o in options" :key="o.value" :value="o.value">{{ o.label }}</option></select>`,
};
const ButtonStub = {
  props: ['label', 'type', 'loading'],
  emits: ['click'],
  template:
    '<button :type="type" :data-loading="loading" @click="$emit(\'click\')">{{ label }}</button>',
};

function mountForm() {
  return mount(TaskSubmitForm, {
    attachTo: document.body,
    global: {
      stubs: {
        Textarea: textInput('textarea'),
        InputText: textInput('input'),
        Select: SelectStub,
        Button: ButtonStub,
      },
    },
  });
}
type W = ReturnType<typeof mountForm>;

const instruction = (w: W) => w.find('textarea');
const field = (w: W, ph: string) => w.find(`input[placeholder^="${ph}"]`);
const select = (w: W, ph: string) => w.find(`select[data-ph^="${ph}"]`);
const submit = (w: W) => w.find('form').trigger('submit');
const menuItem = (w: W, text: string) =>
  w.findAll('[role="menuitem"]').find((b) => b.text().includes(text))!;

beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
  api.submitTask.mockResolvedValue('t-1');
  api.createDraft.mockResolvedValue('d-1');
});

describe('TaskSubmitForm — validation', () => {
  it('requires an instruction', async () => {
    const w = mountForm();
    expect(w.text()).not.toContain('Instruction is required.');
    await submit(w);
    expect(w.text()).toContain('Instruction is required.');
    expect(api.submitTask).not.toHaveBeenCalled();
    w.unmount();
  });

  it('treats a whitespace-only instruction as empty', async () => {
    const w = mountForm();
    await instruction(w).setValue('   ');
    await submit(w);
    expect(api.submitTask).not.toHaveBeenCalled();
    w.unmount();
  });

  it('also validates before saving a draft or backlog item', async () => {
    const w = mountForm();
    await w.find('button[aria-label="More submit options"]').trigger('click');
    await menuItem(w, 'Save as Draft').trigger('click');
    await w.find('button[aria-label="More submit options"]').trigger('click');
    await menuItem(w, 'Add to Backlog').trigger('click');
    expect(api.createDraft).not.toHaveBeenCalled();
    expect(w.text()).toContain('Instruction is required.');
    w.unmount();
  });
});

describe('TaskSubmitForm — submit', () => {
  it('sends every field, including priority and tags', async () => {
    const w = mountForm();
    await instruction(w).setValue('  Add retries  ');
    await field(w, 'Project path').setValue('/p');
    await field(w, 'Target file').setValue('h.go');
    await select(w, 'Command').setValue('plan');
    await select(w, 'Priority').setValue(1);
    const tag = field(w, 'Add tag');
    await tag.setValue('urgent');
    await tag.trigger('keydown.enter');
    await submit(w);
    await flushPromises();

    expect(api.submitTask).toHaveBeenCalledWith({
      projectPath: '/p',
      targetFile: 'h.go',
      instruction: 'Add retries',
      command: 'plan',
      providerHint: '',
      modelId: '',
      contextFiles: [],
      priority: 1,
      tags: ['urgent'],
    });
    w.unmount();
  });

  it('toasts, emits the new id and resets the form on success', async () => {
    const w = mountForm();
    await instruction(w).setValue('Do it');
    await submit(w);
    await flushPromises();
    expect(api.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'success', detail: 'ID: t-1' }),
    );
    expect(w.emitted('submitted')![0]).toEqual(['t-1']);
    expect((instruction(w).element as HTMLTextAreaElement).value).toBe('');
    expect(w.text()).not.toContain('Instruction is required.');
    w.unmount();
  });

  it('toasts the error and keeps the form on failure', async () => {
    api.submitTask.mockRejectedValue(new Error('queue full'));
    const w = mountForm();
    await instruction(w).setValue('Do it');
    await submit(w);
    await flushPromises();
    expect(api.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'error', detail: 'queue full' }),
    );
    expect(w.emitted('submitted')).toBeUndefined();
    expect((instruction(w).element as HTMLTextAreaElement).value).toBe('Do it');
    w.unmount();
  });

  it('stringifies a non-Error failure', async () => {
    api.submitTask.mockRejectedValue('plain');
    const w = mountForm();
    await instruction(w).setValue('Do it');
    await submit(w);
    await flushPromises();
    expect(api.toastAdd).toHaveBeenCalledWith(expect.objectContaining({ detail: 'plain' }));
    w.unmount();
  });

  it('shows progress while submitting', async () => {
    let done!: (id: string) => void;
    api.submitTask.mockReturnValue(new Promise<string>((r) => (done = r)));
    const w = mountForm();
    await instruction(w).setValue('Do it');
    await submit(w);
    expect(w.text()).toContain('Submitting…');
    done('t-1');
    await flushPromises();
    expect(w.text()).toContain('Submit to Queue');
    w.unmount();
  });
});

describe('TaskSubmitForm — provider and model', () => {
  it('lists detected providers and loads the chosen provider’s models', async () => {
    const w = mountForm();
    const model = () => select(w, 'Default model');
    expect(model().attributes('disabled')).toBeDefined();
    expect(select(w, 'Auto').text()).toContain('Ollama');

    await select(w, 'Auto').setValue('Ollama');
    expect(model().attributes('disabled')).toBeUndefined();
    expect(model().text()).toContain('llama3');
    expect(model().text()).toContain('qwen');

    await instruction(w).setValue('x');
    await model().setValue('qwen');
    await submit(w);
    await flushPromises();
    expect(api.submitTask).toHaveBeenCalledWith(
      expect.objectContaining({ providerHint: 'Ollama', modelId: 'qwen' }),
    );
    w.unmount();
  });

  it('offers only the default model for a provider that reports none', async () => {
    const w = mountForm();
    await select(w, 'Auto').setValue('LM Studio');
    expect(select(w, 'Default model').findAll('option')).toHaveLength(1);
    w.unmount();
  });
});

describe('TaskSubmitForm — tags', () => {
  it('adds on Enter and on comma, trims, and ignores blanks and duplicates', async () => {
    const w = mountForm();
    const tag = field(w, 'Add tag');
    await tag.setValue('a');
    await tag.trigger('keydown.enter');
    await tag.setValue('b');
    await tag.trigger('keydown', { key: ',' });
    await tag.setValue('a');
    await tag.trigger('keydown.enter');
    await tag.setValue('   ');
    await tag.trigger('keydown.enter');
    await tag.setValue('x,y');
    await tag.trigger('keydown.enter');
    const chips = w.findAll('span.rounded-full').map((s) => s.text().replace('×', '').trim());
    expect(chips).toEqual(['a', 'b', 'xy']);
    w.unmount();
  });

  it('ignores other keys and removes a tag from its chip', async () => {
    const w = mountForm();
    const tag = field(w, 'Add tag');
    await tag.setValue('a');
    await tag.trigger('keydown', { key: 'x' });
    expect(w.findAll('span.rounded-full')).toHaveLength(0);
    await tag.trigger('keydown.enter');
    await w.find('button[aria-label="Remove tag a"]').trigger('click');
    expect(w.findAll('span.rounded-full')).toHaveLength(0);
    w.unmount();
  });
});

describe('TaskSubmitForm — drafts and backlog', () => {
  async function openMenu(w: W) {
    await w.find('button[aria-label="More submit options"]').trigger('click');
  }

  it('saves a draft with DRAFT status', async () => {
    const w = mountForm();
    await instruction(w).setValue('Later');
    await openMenu(w);
    await menuItem(w, 'Save as Draft').trigger('click');
    await flushPromises();
    expect(api.createDraft).toHaveBeenCalledWith(
      expect.objectContaining({ instruction: 'Later', status: 'DRAFT', priority: 2 }),
    );
    expect(api.toastAdd).toHaveBeenCalledWith(expect.objectContaining({ summary: 'Draft saved' }));
    expect(w.emitted('submitted')![0]).toEqual(['d-1']);
    expect(w.find('[role="menu"]').exists()).toBe(false);
    w.unmount();
  });

  it('adds to the backlog with BACKLOG status', async () => {
    const w = mountForm();
    await instruction(w).setValue('Later');
    await openMenu(w);
    await menuItem(w, 'Add to Backlog').trigger('click');
    await flushPromises();
    expect(api.createDraft).toHaveBeenCalledWith(expect.objectContaining({ status: 'BACKLOG' }));
    expect(api.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ summary: 'Added to backlog' }),
    );
    w.unmount();
  });

  it.each(['Save as Draft', 'Add to Backlog'])(
    '%s reports failures and keeps the form',
    async (label) => {
      api.createDraft.mockRejectedValueOnce(new Error('disk full')).mockRejectedValueOnce('plain');
      const w = mountForm();
      await instruction(w).setValue('Later');
      await openMenu(w);
      await menuItem(w, label).trigger('click');
      await flushPromises();
      expect(api.toastAdd).toHaveBeenLastCalledWith(
        expect.objectContaining({ summary: 'Save failed', detail: 'disk full' }),
      );
      await openMenu(w);
      await menuItem(w, label).trigger('click');
      await flushPromises();
      expect(api.toastAdd).toHaveBeenLastCalledWith(expect.objectContaining({ detail: 'plain' }));
      expect((instruction(w).element as HTMLTextAreaElement).value).toBe('Later');
      w.unmount();
    },
  );

  it('closes the menu on Escape and on an outside click, but not on an inside click', async () => {
    const w = mountForm();
    await openMenu(w);
    await w.find('[role="menu"]').trigger('keydown.esc');
    expect(w.find('[role="menu"]').exists()).toBe(false);

    await openMenu(w);
    document.body.click();
    await flushPromises();
    expect(w.find('[role="menu"]').exists()).toBe(false);

    await openMenu(w);
    await w.find('[role="menu"]').trigger('click');
    document.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    w.unmount();
  });

  it('stops listening for outside clicks after unmount', () => {
    const spy = vi.spyOn(document, 'removeEventListener');
    mountForm().unmount();
    expect(spy).toHaveBeenCalledWith('click', expect.any(Function));
    spy.mockRestore();
  });
});

describe('TaskSubmitForm — chrome', () => {
  it('collapses and expands', async () => {
    const w = mountForm();
    const toggle = w.find('button[aria-label="Collapse form"]');
    await toggle.trigger('click');
    expect(w.find('.space-y-3').attributes('style')).toContain('display: none');
    await w.find('button[aria-label="Expand form"]').trigger('click');
    expect(w.find('.space-y-3').attributes('style') ?? '').not.toContain('display: none');
    w.unmount();
  });

  it('Clear resets every field', async () => {
    const w = mountForm();
    await instruction(w).setValue('x');
    await field(w, 'Project path').setValue('/p');
    await field(w, 'Add tag').setValue('t');
    await w
      .findAll('button')
      .find((b) => b.text() === 'Clear')!
      .trigger('click');
    expect((instruction(w).element as HTMLTextAreaElement).value).toBe('');
    expect((field(w, 'Project path').element as HTMLInputElement).value).toBe('');
    expect((field(w, 'Add tag').element as HTMLInputElement).value).toBe('');
    w.unmount();
  });
});
