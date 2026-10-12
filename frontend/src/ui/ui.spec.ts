import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import UiIcon from './UiIcon.vue';
import UiButton from './UiButton.vue';
import UiSkeleton from './UiSkeleton.vue';
import UiSelect from './UiSelect.vue';
import UiDrawer from './UiDrawer.vue';
import UiToaster from './UiToaster.vue';
import { ICONS } from './icons';
import { activeToasts, useToast } from '../composables/useToast';

describe('UiIcon', () => {
  it('draws every path of the named icon, hidden from assistive technology', () => {
    const w = mount(UiIcon, { props: { name: 'check-circle' } });
    expect(w.findAll('path')).toHaveLength(ICONS['check-circle'].length);
    expect(w.attributes('data-icon')).toBe('check-circle');
    expect(w.attributes('aria-hidden')).toBe('true');
  });

  it('has well-formed path data for every icon', () => {
    for (const [name, paths] of Object.entries(ICONS)) {
      expect(paths.length, name).toBeGreaterThan(0);
      for (const d of paths) expect(d, name).toMatch(/^[Mm]/);
    }
  });
});

describe('UiSkeleton', () => {
  it('uses the given height and defaults to 1rem', () => {
    expect(mount(UiSkeleton, { props: { height: '52px' } }).attributes('style')).toContain('52px');
    expect(mount(UiSkeleton).attributes('style')).toContain('1rem');
  });
});

describe('UiButton', () => {
  it('renders the label as a type=button by default and forwards clicks', async () => {
    const onClick = vi.fn();
    const w = mount(UiButton, { props: { label: 'Go' }, attrs: { onClick } });
    expect(w.text()).toBe('Go');
    expect(w.attributes('type')).toBe('button');
    await w.trigger('click');
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('prefers slot content, shows an icon, and submits when asked', () => {
    const w = mount(UiButton, {
      props: { icon: 'send', type: 'submit' },
      slots: { default: 'Send' },
    });
    expect(w.text()).toBe('Send');
    expect(w.find('[data-icon="send"]').exists()).toBe(true);
    expect(w.attributes('type')).toBe('submit');
  });

  it('replaces the icon with a spinner and disables itself while loading', () => {
    const w = mount(UiButton, { props: { label: 'Go', icon: 'send', loading: true } });
    expect(w.find('[data-icon="spinner"]').exists()).toBe(true);
    expect(w.find('[data-icon="send"]').exists()).toBe(false);
    expect(w.attributes('disabled')).toBeDefined();
    expect(w.attributes('aria-busy')).toBe('true');
  });

  it('is disabled when asked and not busy otherwise', () => {
    const w = mount(UiButton, { props: { label: 'Go', disabled: true } });
    expect(w.attributes('disabled')).toBeDefined();
    expect(w.attributes('aria-busy')).toBeUndefined();
  });

  it.each([
    [{ severity: 'danger', outlined: true }, 'text-red-400'],
    [{ severity: 'danger' }, 'bg-red-600'],
    [{ severity: 'secondary', text: true }, 'border-transparent'],
    [{ outlined: true }, 'text-indigo-300'],
    [{}, 'bg-indigo-600'],
  ] as const)('applies the tone for %j', (props, cls) => {
    expect(
      mount(UiButton, { props: { ...props } })
        .classes()
        .join(' '),
    ).toContain(cls);
  });

  it('supports the small size and rounded shape', () => {
    const w = mount(UiButton, { props: { size: 'small', rounded: true } });
    expect(w.classes()).toEqual(expect.arrayContaining(['text-xs', 'rounded-full']));
    expect(mount(UiButton).classes()).toEqual(expect.arrayContaining(['text-sm', 'rounded-md']));
  });
});

describe('UiSelect', () => {
  const options = [
    { label: 'High', value: 1 },
    { label: 'Auto', value: 'auto' },
  ];
  const mountSelect = (modelValue: string | number, extra = {}) =>
    mount(UiSelect, { props: { modelValue, options, label: 'Priority', ...extra } });

  it('selects the option matching the value and emits typed values', async () => {
    const w = mountSelect(1);
    expect((w.element as HTMLSelectElement).value).toBe('0');
    await w.find('select').setValue('1');
    expect(w.emitted('update:modelValue')![0]).toEqual(['auto']);
    await w.find('select').setValue('0');
    expect(w.emitted('update:modelValue')![1]).toEqual([1]);
  });

  it('shows the placeholder when nothing matches', () => {
    const w = mountSelect('missing', { placeholder: 'Pick one' });
    expect(w.find('option[disabled]').text()).toBe('Pick one');
  });

  it('has an accessible name and reflects disabled and invalid', () => {
    const w = mountSelect(1, { disabled: true, invalid: true });
    expect(w.attributes('aria-label')).toBe('Priority');
    expect(w.attributes('disabled')).toBeDefined();
    expect(w.attributes('aria-invalid')).toBe('true');
    expect(mountSelect(1).attributes('aria-invalid')).toBeUndefined();
  });
});

describe('UiDrawer', () => {
  let opener: HTMLButtonElement;

  beforeEach(() => {
    opener = document.createElement('button');
    document.body.appendChild(opener);
    opener.focus();
  });
  afterEach(() => {
    document.body.innerHTML = '';
  });

  const mountDrawer = (modelValue = true) =>
    mount(UiDrawer, {
      props: { modelValue, label: 'Details' },
      slots: {
        header: '<span>Title</span>',
        default: '<button id="a">A</button><button id="b">B</button>',
      },
      attachTo: document.body,
    });

  it('renders a labelled modal dialog with header and body, and nothing when closed', () => {
    mountDrawer();
    const dialog = document.querySelector('[role="dialog"]')!;
    expect(dialog.getAttribute('aria-modal')).toBe('true');
    expect(dialog.getAttribute('aria-label')).toBe('Details');
    expect(dialog.textContent).toContain('Title');
    document.body.innerHTML = '';
    mountDrawer(false);
    expect(document.querySelector('[role="dialog"]')).toBeNull();
  });

  it('closes from the close button, the backdrop and Escape', async () => {
    const w = mountDrawer();
    (document.querySelector('[aria-label="Close"]') as HTMLElement).click();
    (document.querySelector('.bg-black\\/50') as HTMLElement).click();
    document
      .querySelector('[role="dialog"]')!
      .dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }));
    expect(w.emitted('update:modelValue')).toEqual([[false], [false], [false]]);
  });

  it('moves focus into the panel when opened and back to the opener when closed', async () => {
    const w = mountDrawer(false);
    await w.setProps({ modelValue: true });
    await nextTick();
    await nextTick();
    expect(document.activeElement).toBe(document.querySelector('[role="dialog"]'));
    await w.setProps({ modelValue: false });
    expect(document.activeElement).toBe(opener);
  });

  it('traps Tab and Shift+Tab inside the panel', async () => {
    mountDrawer();
    const key = (shiftKey: boolean) => {
      const e = new KeyboardEvent('keydown', {
        key: 'Tab',
        shiftKey,
        bubbles: true,
        cancelable: true,
      });
      document.querySelector('[role="dialog"]')!.dispatchEvent(e);
      return e;
    };
    const items = [...document.querySelectorAll<HTMLElement>('[role="dialog"] button')];
    const first = items[0]!;
    const last = items[items.length - 1]!;

    last.focus();
    expect(key(false).defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(first);

    first.focus();
    expect(key(true).defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(last);

    (document.querySelector('[role="dialog"]') as HTMLElement).focus();
    expect(key(true).defaultPrevented).toBe(true);

    items[1]!.focus();
    expect(key(false).defaultPrevented).toBe(false);
  });

  it('ignores other keys and keeps focus on the panel when it has nothing focusable', async () => {
    const w = mount(UiDrawer, { props: { modelValue: true }, attachTo: document.body });
    const dialog = document.querySelector('[role="dialog"]') as HTMLElement;
    dialog.dispatchEvent(new KeyboardEvent('keydown', { key: 'a', bubbles: true }));
    expect(w.emitted('update:modelValue')).toBeUndefined();
    document.querySelector<HTMLElement>('[aria-label="Close"]')!.remove();
    dialog.focus();
    const e = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true });
    dialog.dispatchEvent(e);
    expect(e.defaultPrevented).toBe(true);
    expect(document.activeElement).toBe(dialog);
  });
});

describe('useToast / UiToaster', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    activeToasts.value.forEach((t) => useToast().remove(t.id));
  });
  afterEach(() => vi.useRealTimers());

  it('shows a toast and removes it after its life (default 3s)', async () => {
    const w = mount(UiToaster);
    useToast().add({ severity: 'success', summary: 'Saved', detail: 'ID: 1' });
    await nextTick();
    expect(w.text()).toContain('Saved');
    expect(w.text()).toContain('ID: 1');
    expect(w.find('[role="status"]').exists()).toBe(true);
    vi.advanceTimersByTime(2999);
    expect(activeToasts.value).toHaveLength(1);
    vi.advanceTimersByTime(1);
    await nextTick();
    expect(w.text()).not.toContain('Saved');
  });

  it('honours a custom life, announces errors as alerts and omits an absent detail', async () => {
    const w = mount(UiToaster);
    useToast().add({ severity: 'error', summary: 'Boom', life: 5000 });
    await nextTick();
    expect(w.find('[role="alert"]').exists()).toBe(true);
    expect(w.findAll('p')).toHaveLength(1);
    vi.advanceTimersByTime(4999);
    expect(activeToasts.value).toHaveLength(1);
    vi.advanceTimersByTime(1);
    expect(activeToasts.value).toHaveLength(0);
  });

  it('dismisses a toast from its close button', async () => {
    const w = mount(UiToaster);
    useToast().add({ severity: 'info', summary: 'Hello' });
    useToast().add({ severity: 'warn', summary: 'Careful' });
    await nextTick();
    await w.find('[aria-label="Dismiss notification"]').trigger('click');
    expect(w.text()).not.toContain('Hello');
    expect(w.text()).toContain('Careful');
  });
});
