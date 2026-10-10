import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';

const { push, route, connected, lastChecked } = vi.hoisted(() => ({
  push: vi.fn(),
  route: { path: '/' },
  connected: { value: true },
  lastChecked: { value: null as Date | null },
}));
vi.mock('vue-router', () => ({ useRouter: () => ({ push }), useRoute: () => route }));
vi.mock('../composables/useDaemonHealth', () => ({
  useDaemonHealth: () => ({ connected: ref(connected.value), lastChecked: ref(lastChecked.value) }),
}));
vi.mock('../router/routes', () => ({
  default: [
    { name: 'home', path: '/', label: 'Mission Control', icon: 'pi-home', nav: true },
    { name: 'providers', path: '/providers', label: 'Providers', icon: 'pi-server', nav: true },
    { name: 'hidden', path: '/hidden', label: 'Hidden', icon: 'pi-eye-slash', nav: false },
  ],
}));
import AppSidebar from './AppSidebar.vue';

const mountSidebar = () => mount(AppSidebar, { global: { stubs: { ProjectSelector: true } } });

describe('AppSidebar', () => {
  beforeEach(() => {
    push.mockReset();
    route.path = '/';
    connected.value = true;
    lastChecked.value = null;
  });

  it('lists only the routes flagged for navigation', () => {
    const labels = mountSidebar()
      .findAll('nav button')
      .map((b) => b.text());
    expect(labels).toEqual(['Mission Control', 'Providers']);
  });

  it('navigates when an item is clicked', async () => {
    await mountSidebar().findAll('nav button')[1].trigger('click');
    expect(push).toHaveBeenCalledWith('/providers');
  });

  it('highlights the root only on an exact match and other items by prefix', () => {
    route.path = '/providers/openai';
    const [home, providers] = mountSidebar().findAll('nav button');
    expect(home.classes().join(' ')).not.toContain('bg-violet-600/15');
    expect(providers.classes().join(' ')).toContain('bg-violet-600/15');

    route.path = '/';
    const [home2] = mountSidebar().findAll('nav button');
    expect(home2.classes().join(' ')).toContain('bg-violet-600/15');
  });

  it('shows the daemon connection state and a descriptive tooltip', () => {
    let w = mountSidebar();
    expect(w.text()).toContain('connected');
    expect(w.find('[title^="nexus-daemon"]').attributes('title')).toContain(
      'Last checked: not checked yet',
    );

    connected.value = false;
    lastChecked.value = new Date('2026-01-01T10:00:00Z');
    w = mountSidebar();
    expect(w.text()).toContain('offline');
    expect(w.find('[title^="nexus-daemon"]').attributes('title')).toMatch(
      /nexus-daemon: offline\nLast checked: .+/,
    );
  });

  it('renders the logo and the project selector', () => {
    const w = mountSidebar();
    expect(w.findAll('img[alt="nexusOrchestrator"]')).toHaveLength(2);
    expect(w.findComponent({ name: 'ProjectSelector' }).exists()).toBe(true);
  });
});
