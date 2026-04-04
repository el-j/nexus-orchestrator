import { defineConfig, devices } from '@playwright/test';

// Frontend dev server URL. Override with BASE_URL env var.
// Vite dev server for this project runs on :63989 (see vite.config.ts).
const baseURL = process.env.BASE_URL ?? 'http://localhost:63989';

export default defineConfig({
  testDir: './e2e',
  retries: 0,
  reporter: [['html', { outputFolder: 'playwright-report', open: 'never' }]],

  use: {
    baseURL,
    screenshot: 'only-on-failure',
  },

  // Single browser: Chromium headless only
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
