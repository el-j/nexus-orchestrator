import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/**', (route) => route.fulfill({ json: [] }));
});

test.describe('Settings', () => {
  test('settings page loads', async ({ page }) => {
    await page.goto('/');

    // Navigate to Settings via sidebar
    await page
      .locator('aside nav')
      .getByRole('button', { name: /settings/i })
      .click();

    // The Settings view h1 must be visible
    await expect(page.getByRole('heading', { name: /^settings$/i })).toBeVisible();
  });

  test('shows queue cap card', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /settings/i })
      .click();

    // "Queue Settings" section heading is always rendered (shows default cap = 50 without daemon)
    await expect(page.getByText(/queue settings/i)).toBeVisible();
  });
});
