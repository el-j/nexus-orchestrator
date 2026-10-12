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
    icon: 'folder',
    name: 'projects',
    path: '/projects',
    label: 'Projects',
    nav: true,
    component: () => import('../views/ProjectActivityView.vue'),
  },
  {
    // Not a nav item — accessed via "View timeline" button in ProjectActivityView
    name: 'live-activity',
    path: '/projects/live-activity',
    component: () => import('../views/LiveActivityView.vue'),
  },
  {
    icon: 'file',
    name: 'plans',
    path: '/plans',
    label: 'Plans',
    nav: true,
    component: () => import('../views/DiscoveredPlansView.vue'),
  },
  {
    icon: 'clock',
    name: 'history',
    path: '/history',
    label: 'History',
    nav: true,
    component: () => import('../views/HistoryView.vue'),
  },
  {
    icon: 'list',
    name: 'backlog',
    path: '/backlog',
    label: 'Backlog',
    nav: true,
    component: () => import('../views/BacklogView.vue'),
  },
  {
    icon: 'database',
    name: 'brain',
    path: '/brain',
    label: 'Brain',
    nav: true,
    component: () => import('../views/BrainView.vue'),
  },
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
