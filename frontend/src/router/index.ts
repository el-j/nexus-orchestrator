import { createRouter, createWebHistory } from 'vue-router';

import routes from './routes';

// The daemon serves this app under /app/ and says so with a <base href> in index.html; the
// desktop app and the dev server have no such tag and live at "/".
const base = document.querySelector('base')?.getAttribute('href') ?? '/';

const router = createRouter({
  history: createWebHistory(base),
  routes,
});

export default router;
