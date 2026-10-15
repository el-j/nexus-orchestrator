import { describe, expect, it, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { createMemoryHistory, createRouter } from 'vue-router';
import ProjectsLayout from './ProjectsLayout.vue';

async function mountAt(path: string) {
  const View = { template: '<p data-testid="view">view</p>' };
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      {
        path: '/projects',
        component: ProjectsLayout,
        children: [
          { path: '', component: View },
          { path: 'live-activity', component: View },
          { path: 'plans', component: View },
          { path: 'history', component: View },
          { path: 'backlog', component: View },
          { path: 'brain', component: View },
        ],
      },
    ],
  });
  const push = vi.spyOn(router, 'push');
  await router.push(path);
  // The app renders the layout through the root router-view, and the layout renders the tab's view.
  const w = mount({ template: '<router-view />' }, { global: { plugins: [router] } });
  return { w, router, push };
}

describe('ProjectsLayout', () => {
  it('shows the five sections of a project as tabs over the routed view', async () => {
    const { w } = await mountAt('/projects/plans');
    expect(w.findAll('nav button').map((b) => b.text())).toEqual([
      'Activity',
      'Plans',
      'History',
      'Backlog',
      'Brain',
    ]);
    expect(w.find('[data-testid="view"]').exists()).toBe(true);
  });

  it('marks the tab of the current page, and keeps Activity lit on the timeline', async () => {
    const { w } = await mountAt('/projects/brain');
    expect(w.get('[data-testid="project-tab-brain"]').attributes('aria-current')).toBe('page');
    expect(w.get('[data-testid="project-tab-plans"]').attributes('aria-current')).toBeUndefined();
    const live = await mountAt('/projects/live-activity');
    expect(live.w.get('[data-testid="project-tab-activity"]').attributes('aria-current')).toBe(
      'page',
    );
  });

  it('goes to a tab when it is clicked', async () => {
    const { w, push } = await mountAt('/projects');
    await w.get('[data-testid="project-tab-history"]').trigger('click');
    expect(push).toHaveBeenCalledWith('/projects/history');
  });
});
