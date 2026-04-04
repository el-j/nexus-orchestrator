import { test, expect } from '@playwright/test';

/**
 * Mock all /api/** requests so tests run without a live daemon.
 * PLAYWRIGHT_SKIP_DAEMON=true is respected at the test level for
 * assertions that would only be meaningful against real data.
 */
test.beforeEach(async ({ page }) => {
  await page.route('**/api/**', (route) => route.fulfill({ json: [] }));
});

test.describe('Dashboard', () => {
  test('loads dashboard and shows nav', async ({ page }) => {
    await page.goto('/');

    // Sidebar with navigation buttons must render
    const sidebar = page.locator('aside nav');
    await expect(sidebar).toBeVisible();

    // The default view nav item ("Task Queue") must be present
    // Sidebar labels are visible at ≥1024 px; Playwright default viewport is 1280x720
    await expect(sidebar.getByRole('button', { name: /task queue/i })).toBeVisible();

    // Settings nav item is always rendered alongside dashboard
    await expect(sidebar.getByRole('button', { name: /settings/i })).toBeVisible();
  });

  test('task submit form exists', async ({ page }) => {
    await page.goto('/');

    // TaskSubmitForm renders at bottom of dashboard even with no data
    await expect(page.getByText('Submit Task')).toBeVisible();
  });
});
