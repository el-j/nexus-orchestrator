import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
  history: createWebHashHistory('/nexus-orchestrator/'),
  routes: [
    { path: '/', component: HomeView, meta: { title: 'nexus-orchestrator — Local AI Task Orchestrator' } },
    { path: '/downloads', component: () => import('../views/DownloadsView.vue'), meta: { title: 'Downloads — nexus-orchestrator' } },
    { path: '/getting-started', component: () => import('../views/GettingStartedView.vue'), meta: { title: 'Getting Started — nexus-orchestrator' } },
    { path: '/architecture', component: () => import('../views/ArchitectureView.vue'), meta: { title: 'Architecture — nexus-orchestrator' } },
    { path: '/api-reference', component: () => import('../views/ApiReferenceView.vue'), meta: { title: 'API Reference — nexus-orchestrator' } },
    { path: '/mcp-integration', component: () => import('../views/McpIntegrationView.vue'), meta: { title: 'MCP Integration — nexus-orchestrator' } },
  ],
})

router.afterEach((to) => {
  const title = to.meta.title as string
  if (title) document.title = title
})

export default router
