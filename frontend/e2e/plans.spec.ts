import { test, expect } from '@playwright/test';

test.beforeEach(async ({ page }) => {
  await page.route('**/api/**', (route) => route.fulfill({ json: [] }));
});

test.describe('Plans', () => {
  test('discovered plans page loads', async ({ page }) => {
    await page.goto('/');

    // Plans is a tab of the Projects hub
    await page
      .locator('aside nav')
      .getByRole('button', { name: /projects/i })
      .click();
    await page.getByTestId('project-tab-plans').click();

    // DiscoveredPlansView h1 must be visible
    await expect(page.getByRole('heading', { name: /^plans$/i })).toBeVisible();
  });

  test('scan button exists', async ({ page }) => {
    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /projects/i })
      .click();
    await page.getByTestId('project-tab-plans').click();

    // "Scan Now" button is always rendered in the Plans header
    await expect(page.getByRole('button', { name: /scan now/i })).toBeVisible();
  });
});
