import type { RouteRecordRaw } from 'vue-router';

export type AppRoute = RouteRecordRaw & {
  label?: string;
  icon?: string;
  /** Show this route as a sidebar nav item */
  nav?: boolean;
};

const routes: AppRoute[] = [
  {
    icon: 'pi-home',
    name: 'mission-control',
    path: '/',
    label: 'Mission Control',
    nav: true,
    component: () => import('../views/MissionControlView.vue'),
  },
  {
    icon: 'pi-users',
    name: 'agents',
    path: '/agents',
    label: 'Agents',
    nav: true,
    component: () => import('../views/AgentsView.vue'),
  },
  {
    icon: 'pi-server',
    name: 'providers',
    path: '/providers',
    label: 'Providers',
    nav: true,
    component: () => import('../views/ProvidersView.vue'),
  },
  {
    icon: 'pi-folder',
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
    icon: 'pi-file',
    name: 'plans',
    path: '/plans',
    label: 'Plans',
    nav: true,
    component: () => import('../views/DiscoveredPlansView.vue'),
  },
  {
    icon: 'pi-clock',
    name: 'history',
    path: '/history',
    label: 'History',
    nav: true,
    component: () => import('../views/HistoryView.vue'),
  },
  {
    icon: 'pi-list',
    name: 'backlog',
    path: '/backlog',
    label: 'Backlog',
    nav: true,
    component: () => import('../views/BacklogView.vue'),
  },
  {
    icon: 'pi-database',
    name: 'brain',
    path: '/brain',
    label: 'Brain',
    nav: true,
    component: () => import('../views/BrainView.vue'),
  },
  {
    icon: 'pi-search',
    name: 'discovery',
    path: '/discovery',
    label: 'Discovery',
    nav: true,
    component: () => import('../views/DiscoveryView.vue'),
  },
  {
    icon: 'pi-desktop',
    name: 'ai-sessions',
    path: '/ai-sessions',
    label: 'AI Sessions',
    nav: true,
    component: () => import('../views/AISessionsView.vue'),
  },
  {
    icon: 'pi-cog',
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
