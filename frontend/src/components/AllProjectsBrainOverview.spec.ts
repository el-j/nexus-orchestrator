import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import AllProjectsBrainOverview from './AllProjectsBrainOverview.vue';
import * as wails from '../types/wails';

vi.mock('../types/wails', () => ({
  getBrainStatus: vi.fn(),
}));

const mockStatus1 = {
  initialized: true,
  entryCount: 42,
  totalTokens: 10500,
  lastUpdated: '2026-11-20T10:00:00Z',
  kindCounts: { architecture: 10, learning: 32 },
};

const mockStatus2 = {
  initialized: false,
  entryCount: 0,
  totalTokens: 0,
  lastUpdated: '',
  kindCounts: {},
};

describe('AllProjectsBrainOverview', () => {
  beforeEach(() => {
    vi.mocked(wails.getBrainStatus).mockReset();
  });

  it('renders empty state when projectList is empty', async () => {
    const wrapper = mount(AllProjectsBrainOverview, {
      props: { projectList: [] },
    });
    await flushPromises();
    expect(wrapper.text()).toContain('No projects discovered in workspace');
  });

  it('fetches status for each project and displays workspace totals', async () => {
    vi.mocked(wails.getBrainStatus).mockImplementation((path: string) => {
      if (path === '/repo/alpha') return Promise.resolve(mockStatus1 as never);
      return Promise.resolve(mockStatus2 as never);
    });

    const wrapper = mount(AllProjectsBrainOverview, {
      props: { projectList: ['/repo/alpha', '/repo/beta'] },
    });
    await flushPromises();

    expect(wails.getBrainStatus).toHaveBeenCalledWith('/repo/alpha');
    expect(wails.getBrainStatus).toHaveBeenCalledWith('/repo/beta');

    // Totals in banner
    expect(wrapper.text()).toContain('1 of 2 projects indexed');
    expect(wrapper.text()).toContain('42 entries');
    expect(wrapper.text()).toContain('10,500 tokens');

    // Cards
    const alphaCard = wrapper.find('[data-testid="brain-card-alpha"]');
    expect(alphaCard.exists()).toBe(true);
    expect(alphaCard.text()).toContain('indexed');
    expect(alphaCard.text()).toContain('42 entries');

    const betaCard = wrapper.find('[data-testid="brain-card-beta"]');
    expect(betaCard.exists()).toBe(true);
    expect(betaCard.text()).toContain('unindexed');
    expect(betaCard.text()).toContain('No knowledge indexed yet');
  });

  it('emits selectProject when clicking Open Brain button', async () => {
    vi.mocked(wails.getBrainStatus).mockResolvedValue(mockStatus1 as never);

    const wrapper = mount(AllProjectsBrainOverview, {
      props: { projectList: ['/repo/alpha'] },
    });
    await flushPromises();

    const openBtn = wrapper.find('button');
    expect(openBtn.text()).toContain('Open Brain →');
    await openBtn.trigger('click');

    expect(wrapper.emitted('selectProject')).toEqual([['/repo/alpha']]);
  });

  it('handles getBrainStatus rejections gracefully', async () => {
    vi.mocked(wails.getBrainStatus).mockRejectedValue(new Error('Network error'));

    const wrapper = mount(AllProjectsBrainOverview, {
      props: { projectList: ['/repo/error-proj'] },
    });
    await flushPromises();

    expect(wrapper.text()).toContain('error-proj');
    expect(wrapper.text()).toContain('0 of 1 projects indexed');
  });
});
