import { createRouter, createMemoryHistory, type Router } from 'vue-router';
import { routes } from '../router';

/** A real router over the site's routes, with in-memory history and stub pages. */
export function memoryRouter(initial = '/'): Router {
  const stub = { template: '<div class="page" />' };
  const router = createRouter({
    history: createMemoryHistory(),
    routes: routes.map((r) => ({ ...r, component: stub })),
  });
  void router.push(initial);
  return router;
}
