import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/**', (route) => route.fulfill({ json: [] }));
});

test.describe('Backlog', () => {
  test('backlog view loads', async ({ page }) => {
    await page.goto('/');

    // Navigate to Backlog via sidebar
    await page
      .locator('aside nav')
      .getByRole('button', { name: /^backlog$/i })
      .click();

    // Backlog view h1 must appear — confirms view switched and rendered without error
    await expect(page.getByRole('heading', { name: /^backlog$/i })).toBeVisible();
  });
});
