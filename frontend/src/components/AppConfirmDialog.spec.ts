import { describe, it, expect, afterEach } from 'vitest';
import { mount, type VueWrapper } from '@vue/test-utils';
import AppConfirmDialog from './AppConfirmDialog.vue';

let wrapper: VueWrapper | undefined;
afterEach(() => wrapper?.unmount());

function open(props: Record<string, unknown> = {}) {
  wrapper = mount(AppConfirmDialog, {
    props: { open: true, title: 'Delete?', message: 'This cannot be undone.', ...props },
    global: { stubs: { Teleport: true, Transition: false } },
  });
  return wrapper;
}

describe('AppConfirmDialog', () => {
  it('renders nothing while closed', () => {
    expect(open({ open: false }).find('[role="dialog"]').exists()).toBe(false);
  });

  it('shows the title and message and the default labels', () => {
    const w = open();
    expect(w.text()).toContain('Delete?');
    expect(w.text()).toContain('This cannot be undone.');
    const buttons = w.findAll('button').map((b) => b.text());
    expect(buttons).toEqual(['Cancel', 'Confirm']);
  });

  it('uses custom labels and the danger styling', () => {
    const w = open({ confirmLabel: 'Remove', cancelLabel: 'Keep', danger: true });
    expect(w.findAll('button').map((b) => b.text())).toEqual(['Keep', 'Remove']);
    expect(w.findAll('button')[1].classes().join(' ')).toContain('red');
  });

  it('emits confirm and cancel from the buttons', async () => {
    const w = open();
    const [cancel, confirm] = w.findAll('button');
    await confirm.trigger('click');
    await cancel.trigger('click');
    expect(w.emitted('confirm')).toHaveLength(1);
    expect(w.emitted('cancel')).toHaveLength(1);
  });

  it('cancels when the backdrop is clicked but not when the panel is', async () => {
    const w = open();
    await w.find('[role="dialog"]').trigger('click');
    expect(w.emitted('cancel')).toBeUndefined();
    await w.find('.fixed').trigger('click');
    expect(w.emitted('cancel')).toHaveLength(1);
  });

  it('cancels on Escape only while open, and stops listening afterwards', async () => {
    const w = open({ open: false });
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(w.emitted('cancel')).toBeUndefined();

    await w.setProps({ open: true });
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter' }));
    expect(w.emitted('cancel')).toBeUndefined();
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(w.emitted('cancel')).toHaveLength(1);

    await w.setProps({ open: false });
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(w.emitted('cancel')).toHaveLength(1);

    await w.setProps({ open: true });
    w.unmount();
    wrapper = undefined;
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
  });
});
