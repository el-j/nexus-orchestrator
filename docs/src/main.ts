import { ViteSSG } from 'vite-ssg';
import './assets/main.css';
import App from './App.vue';
import { routes } from './router';

// ViteSSG exports a factory consumed by the vite-ssg CLI at build time.
// During SSG the factory is called once per route with createMemoryHistory.
// During client hydration the factory is called once with createWebHistory.
// import.meta.env.BASE_URL is replaced at build time with vite.config.ts `base`
// ('/nexus-orchestrator/'), ensuring RouterLink hrefs are fully-qualified
// relative to the GitHub Pages deployment root.
export const createApp = ViteSSG(
  App,
  { routes, base: import.meta.env.BASE_URL },
  ({ router, isClient }) => {
    // Update document title on navigation — client-only because document is
    // not available during server-side pre-rendering.
    if (isClient) {
      router.afterEach((to) => {
        const title = to.meta?.title as string | undefined;
        if (title) document.title = title;
      });
    }
  },
);
