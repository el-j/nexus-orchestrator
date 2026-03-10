import { createRouter, createWebHashHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
  history: createWebHashHistory('/nexusOrchestrator/'),
  routes: [
    { path: '/', component: HomeView, meta: { title: 'nexusOrchestrator — Local AI Task Orchestrator' } },
    { path: '/downloads', component: () => import('../views/DownloadsView.vue'), meta: { title: 'Downloads — nexusOrchestrator' } },
    { path: '/getting-started', component: () => import('../views/GettingStartedView.vue'), meta: { title: 'Getting Started — nexusOrchestrator' } },
    { path: '/architecture', component: () => import('../views/ArchitectureView.vue'), meta: { title: 'Architecture — nexusOrchestrator' } },
    { path: '/api-reference', component: () => import('../views/ApiReferenceView.vue'), meta: { title: 'API Reference — nexusOrchestrator' } },
    { path: '/mcp-integration', component: () => import('../views/McpIntegrationView.vue'), meta: { title: 'MCP Integration — nexusOrchestrator' } },
  ],
})

router.afterEach((to) => {
  const title = to.meta.title as string
  if (title) document.title = title
})

export default router
