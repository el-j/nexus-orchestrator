import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { memoryRouter } from './test/router';
import App from './App.vue';

type IOCallback = (entries: Array<{ isIntersecting: boolean; target: Element }>) => void;
let callback: IOCallback;
const observe = vi.fn();
const unobserve = vi.fn();

beforeEach(() => {
  observe.mockReset();
  unobserve.mockReset();
  vi.stubGlobal(
    'IntersectionObserver',
    class {
      constructor(cb: IOCallback) {
        callback = cb;
      }
      observe = observe;
      unobserve = unobserve;
    },
  );
  document.body.innerHTML = '';
});
afterEach(() => vi.unstubAllGlobals());

const stubs = { AppNav: true, AppFooter: true, Toast: true };

async function mountApp() {
  const router = memoryRouter('/');
  await router.isReady();
  return {
    w: mount(App, { attachTo: document.body, global: { plugins: [router], stubs } }),
    router,
  };
}

describe('App', () => {
  it('renders the shell around the routed page', async () => {
    const { w } = await mountApp();
    expect(w.findComponent({ name: 'AppNav' }).exists()).toBe(true);
    expect(w.findComponent({ name: 'AppFooter' }).exists()).toBe(true);
    expect(w.find('main .page').exists()).toBe(true);
  });

  it('observes elements that still need to reveal, and reveals them once visible', async () => {
    const el = document.createElement('div');
    el.className = 'reveal';
    document.body.appendChild(el);
    await mountApp();
    expect(observe).toHaveBeenCalledWith(el);

    callback([{ isIntersecting: false, target: el }]);
    expect(el.classList.contains('visible')).toBe(false);
    callback([{ isIntersecting: true, target: el }]);
    expect(el.classList.contains('visible')).toBe(true);
    expect(unobserve).toHaveBeenCalledWith(el);
  });

  it('does not re-observe elements that already became visible', async () => {
    const done = document.createElement('div');
    done.className = 'reveal visible';
    document.body.appendChild(done);
    await mountApp();
    expect(observe).not.toHaveBeenCalled();
  });

  it('picks up reveal elements rendered by a later navigation', async () => {
    const { router } = await mountApp();
    observe.mockClear();
    const fresh = document.createElement('div');
    fresh.className = 'reveal';
    document.body.appendChild(fresh);
    await router.push('/downloads');
    await new Promise((r) => setTimeout(r, 0));
    expect(observe).toHaveBeenCalledWith(fresh);
  });
});
