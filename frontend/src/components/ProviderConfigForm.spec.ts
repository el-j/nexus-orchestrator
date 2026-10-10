import { describe, it, expect, vi } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import ProviderConfigForm from './ProviderConfigForm.vue';
import type { ProviderConfig } from '../types/domain';

const existing = (o: Partial<ProviderConfig> = {}): ProviderConfig => ({
  id: 'c1',
  name: 'Mine',
  kind: 'ollama',
  baseUrl: 'http://127.0.0.1:11434',
  apiKey: '',
  model: 'llama3',
  enabled: true,
  createdAt: '',
  updatedAt: '',
  ...o,
});

function mountForm(
  modelValue: ProviderConfig | null = null,
  onSave = vi.fn().mockResolvedValue(undefined),
) {
  const onClose = vi.fn();
  const w = mount(ProviderConfigForm, {
    props: { modelValue, onClose, onSave },
    attachTo: document.body,
  });
  return { w, onClose, onSave };
}

const inputs = (w: ReturnType<typeof mountForm>['w']) => w.findAll('input');
const [NAME, URL, KEY, MODEL] = [0, 1, 2, 3];

describe('ProviderConfigForm', () => {
  it('starts as an empty LM Studio form when adding', () => {
    const { w } = mountForm();
    expect(w.text()).toContain('Add Provider');
    expect((w.find('[data-testid="provider-type"]').element as HTMLSelectElement).value).toBe(
      'lmstudio',
    );
    expect((inputs(w)[URL].element as HTMLInputElement).value).toBe('http://127.0.0.1:1234/v1');
    expect((inputs(w)[NAME].element as HTMLInputElement).value).toBe('');
    w.unmount();
  });

  it('pre-fills from an existing config when editing', () => {
    const { w } = mountForm(existing());
    expect(w.text()).toContain('Edit Provider');
    expect((w.find('[data-testid="provider-type"]').element as HTMLSelectElement).value).toBe(
      'ollama',
    );
    expect((inputs(w)[NAME].element as HTMLInputElement).value).toBe('Mine');
    expect((inputs(w)[MODEL].element as HTMLInputElement).value).toBe('llama3');
    w.unmount();
  });

  it('shows an OpenAI-pointing openaicompat config as "OpenAI", other URLs as Custom', () => {
    const a = mountForm(existing({ kind: 'openaicompat', baseUrl: 'https://api.openai.com/v1' }));
    expect((a.w.find('[data-testid="provider-type"]').element as HTMLSelectElement).value).toBe(
      'openai',
    );
    a.w.unmount();
    const b = mountForm(existing({ kind: 'openaicompat', baseUrl: 'http://gateway.local/v1' }));
    expect((b.w.find('[data-testid="provider-type"]').element as HTMLSelectElement).value).toBe(
      'openaicompat',
    );
    b.w.unmount();
  });

  it('switching the type loads that preset URL', async () => {
    const { w } = mountForm();
    const select = w.find('[data-testid="provider-type"]');
    await select.setValue('anthropic');
    expect((inputs(w)[URL].element as HTMLInputElement).value).toBe('https://api.anthropic.com');
    await select.setValue('gemini');
    expect((inputs(w)[URL].element as HTMLInputElement).value).toBe(
      'https://generativelanguage.googleapis.com',
    );
    await select.setValue('openaicompat');
    expect((inputs(w)[URL].element as HTMLInputElement).value).toBe('');
    w.unmount();
  });

  it('saves the OpenAI preset as the openaicompat kind the daemon understands', async () => {
    const { w, onSave } = mountForm();
    await w.find('[data-testid="provider-type"]').setValue('openai');
    await inputs(w)[NAME].setValue('GPT');
    await inputs(w)[KEY].setValue('sk-test');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(onSave).toHaveBeenCalledTimes(1);
    expect(onSave.mock.calls[0][0]).toMatchObject({
      kind: 'openaicompat',
      name: 'GPT',
      baseUrl: 'https://api.openai.com/v1',
      apiKey: 'sk-test',
    });
    w.unmount();
  });

  it('keeps native kinds such as gemini unchanged', async () => {
    const { w, onSave } = mountForm();
    await w.find('[data-testid="provider-type"]').setValue('gemini');
    await inputs(w)[NAME].setValue('G');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(onSave.mock.calls[0][0].kind).toBe('gemini');
    w.unmount();
  });

  it('rejects a blank name without calling onSave', async () => {
    const { w, onSave } = mountForm();
    await inputs(w)[NAME].setValue('   ');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(w.text()).toContain('Name is required.');
    expect(onSave).not.toHaveBeenCalled();
    w.unmount();
  });

  it('rejects a base URL that is not http(s)', async () => {
    const { w, onSave } = mountForm();
    await inputs(w)[NAME].setValue('X');
    await inputs(w)[URL].setValue('ftp://host');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(w.text()).toContain('Base URL must start with http:// or https://');
    expect(onSave).not.toHaveBeenCalled();
    w.unmount();
  });

  it('accepts an empty base URL (the daemon applies its default)', async () => {
    const { w, onSave } = mountForm();
    await inputs(w)[NAME].setValue('X');
    await inputs(w)[URL].setValue('');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(onSave).toHaveBeenCalledTimes(1);
    w.unmount();
  });

  it('surfaces an Error thrown by onSave and re-enables the button', async () => {
    const { w } = mountForm(null, vi.fn().mockRejectedValue(new Error('boom')));
    await inputs(w)[NAME].setValue('X');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(w.text()).toContain('boom');
    expect(w.find('button[type="submit"]').attributes('disabled')).toBeUndefined();
    w.unmount();
  });

  it('stringifies a non-Error rejection', async () => {
    const { w } = mountForm(null, vi.fn().mockRejectedValue('plain'));
    await inputs(w)[NAME].setValue('X');
    await w.find('form').trigger('submit');
    await flushPromises();
    expect(w.text()).toContain('plain');
    w.unmount();
  });

  it('disables Save and shows progress while saving', async () => {
    let finish!: () => void;
    const onSave = vi.fn(() => new Promise<void>((r) => (finish = r)));
    const { w } = mountForm(null, onSave);
    await inputs(w)[NAME].setValue('X');
    await w.find('form').trigger('submit');
    const btn = w.find('button[type="submit"]');
    expect(btn.text()).toBe('Saving…');
    expect(btn.attributes('disabled')).toBeDefined();
    finish();
    await flushPromises();
    expect(btn.text()).toBe('Save');
    w.unmount();
  });

  it('closes via Cancel, backdrop click and the Escape key', async () => {
    const { w, onClose } = mountForm();
    await w
      .findAll('button')
      .find((b) => b.text() === 'Cancel')!
      .trigger('click');
    expect(onClose).toHaveBeenCalledTimes(1);

    await w.find('.fixed').trigger('click'); // backdrop (click.self)
    expect(onClose).toHaveBeenCalledTimes(2);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onClose).toHaveBeenCalledTimes(3);

    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'a' }));
    expect(onClose).toHaveBeenCalledTimes(3);
    w.unmount();
  });

  it('does not close when clicking inside the dialog', async () => {
    const { w, onClose } = mountForm();
    await w.find('[role="dialog"]').trigger('click');
    expect(onClose).not.toHaveBeenCalled();
    w.unmount();
  });

  it('stops listening for Escape after unmount', () => {
    const { w, onClose } = mountForm();
    w.unmount();
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onClose).not.toHaveBeenCalled();
  });

  it('resets the form and clears the error when modelValue changes', async () => {
    const { w } = mountForm(existing());
    await inputs(w)[NAME].setValue('');
    await w.find('form').trigger('submit');
    expect(w.text()).toContain('Name is required.');

    await w.setProps({
      modelValue: existing({
        name: 'Other',
        kind: 'anthropic',
        baseUrl: 'https://api.anthropic.com',
      }),
    });
    expect(w.text()).not.toContain('Name is required.');
    expect((inputs(w)[NAME].element as HTMLInputElement).value).toBe('Other');
    expect((w.find('[data-testid="provider-type"]').element as HTMLSelectElement).value).toBe(
      'anthropic',
    );

    await w.setProps({ modelValue: null });
    expect(w.text()).toContain('Add Provider');
    expect((inputs(w)[NAME].element as HTMLInputElement).value).toBe('');
    w.unmount();
  });
});
