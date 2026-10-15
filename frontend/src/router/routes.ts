import type { RouteRecordRaw } from 'vue-router';
import type { IconName } from '../ui/icons';

/** A route entry of the app shell: path, title and view. */
export type AppRoute = RouteRecordRaw & {
  label?: string;
  icon?: IconName;
  /** Show this route as a sidebar nav item */
  nav?: boolean;
};

const routes: AppRoute[] = [
  {
    icon: 'home',
    name: 'mission-control',
    path: '/',
    label: 'Mission Control',
    nav: true,
    component: () => import('../views/MissionControlView.vue'),
  },
  {
    icon: 'users',
    name: 'agents',
    path: '/agents',
    label: 'Agents',
    nav: true,
    component: () => import('../views/AgentsView.vue'),
  },
  {
    icon: 'server',
    name: 'providers',
    path: '/providers',
    label: 'Providers',
    nav: true,
    component: () => import('../views/ProvidersView.vue'),
  },
  {
    // Everything that belongs to a project, as tabs of one hub: activity, plans, history, backlog, brain.
    icon: 'folder',
    path: '/projects',
    label: 'Projects',
    nav: true,
    component: () => import('../views/ProjectsLayout.vue'),
    children: [
      {
        name: 'projects',
        path: '',
        component: () => import('../views/ProjectActivityView.vue'),
      },
      {
        // Not a tab: opened with "View timeline" from the activity tab.
        name: 'live-activity',
        path: 'live-activity',
        component: () => import('../views/LiveActivityView.vue'),
      },
      { name: 'plans', path: 'plans', component: () => import('../views/DiscoveredPlansView.vue') },
      { name: 'history', path: 'history', component: () => import('../views/HistoryView.vue') },
      { name: 'backlog', path: 'backlog', component: () => import('../views/BacklogView.vue') },
      { name: 'brain', path: 'brain', component: () => import('../views/BrainView.vue') },
    ],
  },
  // The old top-level pages now live in the Projects hub.
  { path: '/plans', redirect: '/projects/plans' },
  { path: '/history', redirect: '/projects/history' },
  { path: '/backlog', redirect: '/projects/backlog' },
  { path: '/brain', redirect: '/projects/brain' },
  {
    icon: 'cog',
    name: 'settings',
    path: '/settings',
    label: 'Settings',
    nav: true,
    component: () => import('../views/SettingsView.vue'),
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: '/',
  },
];

export default routes;
