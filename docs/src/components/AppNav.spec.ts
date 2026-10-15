import { describe, it, expect, vi, afterEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { memoryRouter } from '../test/router';

const DrawerStub = {
  props: ['modelValue'],
  emits: ['update:modelValue'],
  template: '<aside v-if="modelValue" data-testid="drawer"><slot name="header" /><slot /></aside>',
};

async function mountNav() {
  const router = memoryRouter();
  await router.isReady();
  const { default: AppNav } = await import('./AppNav.vue');
  return mount(AppNav, { global: { plugins: [router], stubs: { UiDrawer: DrawerStub } } });
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.resetModules();
});

describe('AppNav (stable channel)', () => {
  it('links every page in order', async () => {
    const w = await mountNav();
    const labels = w.findAll('ul a').map((a) => a.text());
    expect(labels).toEqual([
      'Home',
      'Architecture',
      'Downloads',
      'API Reference',
      'Getting Started',
      'Guides',
      'MCP Integration',
    ]);
  });

  it('offers a switch to the beta docs and no beta banner', async () => {
    const w = await mountNav();
    expect(w.text()).not.toContain('beta / develop');
    expect(w.get('a[title="Switch to beta docs"]').attributes('href')).toBe(
      '/nexus-orchestrator/beta/',
    );
    expect(w.find('a[title="Switch to stable docs"]').exists()).toBe(false);
    expect(w.get('nav').classes()).toContain('top-0');
  });

  it('links the repository and the download page', async () => {
    const w = await mountNav();
    expect(w.get('a[href="https://github.com/el-j/nexus-orchestrator"]').exists()).toBe(true);
    expect(w.findAll('a').some((a) => a.text() === 'Download')).toBe(true);
  });

  it('closes the drawer when it asks to be dismissed', async () => {
    const w = await mountNav();
    await w.get('button[aria-label="Open menu"]').trigger('click');
    w.findComponent(DrawerStub).vm.$emit('update:modelValue', false);
    await flushPromises();
    expect(w.find('[data-testid="drawer"]').exists()).toBe(false);
  });

  it('opens the mobile drawer from the menu button and closes it after navigating', async () => {
    const w = await mountNav();
    expect(w.find('[data-testid="drawer"]').exists()).toBe(false);
    await w.get('button[aria-label="Open menu"]').trigger('click');
    const drawer = w.get('[data-testid="drawer"]');
    expect(drawer.text()).toContain('Home');
    await drawer.findAll('nav a')[1].trigger('click');
    await flushPromises();
    expect(w.find('[data-testid="drawer"]').exists()).toBe(false);
  });
});

describe('AppNav (beta channel)', () => {
  it('shows the banner, a way back to stable, and a taller spacer', async () => {
    vi.stubEnv('VITE_DOCS_CHANNEL', 'beta');
    vi.stubEnv('VITE_STABLE_URL', '/stable/');
    const w = await mountNav();
    expect(w.text()).toContain('beta / develop');
    expect(w.get('a[title="Switch to stable docs"]').attributes('href')).toBe('/stable/');
    expect(w.find('a[title="Switch to beta docs"]').exists()).toBe(false);
    expect(w.get('nav').classes()).toContain('top-8');
    expect(w.find('.h-24').exists()).toBe(true);
  });
});
