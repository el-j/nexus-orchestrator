import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { nextTick } from 'vue';
import UiIcon from './UiIcon.vue';
import UiDrawer from './UiDrawer.vue';
import UiTabs from './UiTabs.vue';
import { ICONS } from './icons';

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

describe('UiTabs', () => {
  const mountTabs = () =>
    mount(UiTabs, {
      props: { tabs: ['One', 'Two', 'Three'], label: 'Sections' },
      slots: { 'panel-0': 'first', 'panel-1': 'second', 'panel-2': 'third' },
      attachTo: document.body,
    });
  const tabs = (w: ReturnType<typeof mountTabs>) => w.findAll('[role="tab"]');
  const press = async (w: ReturnType<typeof mountTabs>, key: string) => {
    await w.get('[role="tablist"]').trigger('keydown', { key });
    await nextTick();
  };

  it('renders an accessible tablist whose first tab is selected', () => {
    const w = mountTabs();
    expect(w.get('[role="tablist"]').attributes('aria-label')).toBe('Sections');
    expect(tabs(w).map((t) => t.attributes('aria-selected'))).toEqual(['true', 'false', 'false']);
    expect(tabs(w).map((t) => t.attributes('tabindex'))).toEqual(['0', '-1', '-1']);
    const panels = w.findAll('[role="tabpanel"]');
    expect(panels.map((p) => p.attributes('hidden') !== undefined)).toEqual([false, true, true]);
    // every panel stays in the DOM so pre-rendered pages keep their content
    expect(w.text()).toContain('third');
    w.unmount();
  });

  it('links each tab to its panel', () => {
    const w = mountTabs();
    const tab = tabs(w)[1]!;
    const panel = w.findAll('[role="tabpanel"]')[1]!;
    expect(tab.attributes('aria-controls')).toBe(panel.attributes('id'));
    expect(panel.attributes('aria-labelledby')).toBe(tab.attributes('id'));
    w.unmount();
  });

  it('selects a tab on click', async () => {
    const w = mountTabs();
    await tabs(w)[2]!.trigger('click');
    expect(tabs(w)[2]!.attributes('aria-selected')).toBe('true');
    expect(w.findAll('[role="tabpanel"]')[2]!.attributes('hidden')).toBeUndefined();
    w.unmount();
  });

  it('moves with the arrow keys, wrapping around, and focuses the new tab', async () => {
    const w = mountTabs();
    await press(w, 'ArrowRight');
    expect(tabs(w)[1]!.attributes('aria-selected')).toBe('true');
    expect(document.activeElement).toBe(tabs(w)[1]!.element);
    await press(w, 'ArrowLeft');
    await press(w, 'ArrowLeft');
    expect(tabs(w)[2]!.attributes('aria-selected')).toBe('true');
    await press(w, 'ArrowRight');
    expect(tabs(w)[0]!.attributes('aria-selected')).toBe('true');
    w.unmount();
  });

  it('jumps with Home and End and ignores other keys', async () => {
    const w = mountTabs();
    await press(w, 'End');
    expect(tabs(w)[2]!.attributes('aria-selected')).toBe('true');
    await press(w, 'Home');
    expect(tabs(w)[0]!.attributes('aria-selected')).toBe('true');
    await press(w, 'a');
    expect(tabs(w)[0]!.attributes('aria-selected')).toBe('true');
    w.unmount();
  });
});
