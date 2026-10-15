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

  // Single browser: Chromium headless only.
  //  - chromium: structure checks with the API mocked (no daemon needed).
  //  - real-daemon: drives the built nexus-daemon plus a fake Ollama, no mocks.
  projects: [
    {
      name: 'chromium',
      testIgnore: /real\//,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'real-daemon',
      testMatch: /real\/.*\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
  ],

  // The real-daemon project needs the daemon stack and the dev server; the mocked
  // project only needs the dev server.
  webServer: [
    ...(process.env.NEXUS_REAL_DAEMON
      ? [
          {
            command: 'node e2e/support/stack.mjs',
            url: 'http://127.0.0.1:63987/api/health',
            reuseExistingServer: !process.env.CI,
            timeout: 60_000,
          },
        ]
      : []),
    {
      command: 'npm run dev',
      url: baseURL,
      reuseExistingServer: !process.env.CI,
      timeout: 60_000,
    },
  ],
});
