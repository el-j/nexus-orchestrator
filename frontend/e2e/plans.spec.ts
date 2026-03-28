import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/**', (route) => route.fulfill({ json: [] }));
});

test.describe('Plans', () => {
  test('discovered plans page loads', async ({ page }) => {
    await page.goto('/');

    // Navigate to Plans via sidebar
    await page
      .locator('aside nav')
      .getByRole('button', { name: /^plans$/i })
      .click();

    // DiscoveredPlansView h1 must be visible
    await expect(page.getByRole('heading', { name: /^plans$/i })).toBeVisible();
  });

  test('scan button exists', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /^plans$/i })
      .click();

    // "Scan Now" button is always rendered in the Plans header
    await expect(page.getByRole('button', { name: /scan now/i })).toBeVisible();
  });
});
