import type { RouteRecordRaw } from 'vue-router';
import HomeView from '../views/HomeView.vue';
import { guides } from '../data/guides';

// This file exports the route definitions array rather than a router instance.
// Previously it exported `export default router` (a `createRouter(...)` result),
// but vite-ssg requires control over router instantiation so it can switch
// between createWebHistory (client hydration) and createMemoryHistory (SSG
// server render) at build time. The routes array is consumed by ViteSSG() in
// main.ts which creates the router internally.
export const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: HomeView,
    meta: { title: 'nexus-orchestrator — Local AI Task Orchestrator' },
  },
  {
    path: '/downloads',
    component: () => import('../views/DownloadsView.vue'),
    meta: { title: 'Downloads — nexus-orchestrator' },
  },
  {
    path: '/getting-started',
    component: () => import('../views/GettingStartedView.vue'),
    meta: { title: 'Getting Started — nexus-orchestrator' },
  },
  {
    path: '/architecture',
    component: () => import('../views/ArchitectureView.vue'),
    meta: { title: 'Architecture — nexus-orchestrator' },
  },
  {
    path: '/api-reference',
    component: () => import('../views/ApiReferenceView.vue'),
    meta: { title: 'API Reference — nexus-orchestrator' },
  },
  {
    path: '/mcp-integration',
    component: () => import('../views/McpIntegrationView.vue'),
    meta: { title: 'MCP Integration — nexus-orchestrator' },
  },
  {
    path: '/guides',
    component: () => import('../views/GuidesView.vue'),
    meta: { title: 'Guides — nexus-orchestrator' },
  },
  // One page per guide (a markdown file of docs/), so the static site has a real page for each.
  ...guides.map((g): RouteRecordRaw => ({
    path: `/guides/${g.slug}`,
    component: () => import('../views/GuideView.vue'),
    meta: { title: `${g.title} — nexus-orchestrator`, slug: g.slug },
  })),
];
