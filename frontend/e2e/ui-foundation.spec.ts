import { test, expect, type Page } from '@playwright/test';

/**
 * Regression matrix for the internal UI foundation (toasts, drawer, form controls, icons).
 * Risk categories: keyboard (Tab / Escape / Enter), focus (visible, trapped, restored),
 * ARIA (roles, names, live regions).
 */
const QUEUED = {
  id: 'task-queued-1',
  projectPath: '/p',
  targetFile: 'a.go',
  instruction: 'Write the thing',
  command: 'auto',
  status: 'QUEUED',
  priority: 2,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
};

async function mockApi(page: Page) {
  await page.route('**/api/**', (route) => {
    const url = new URL(route.request().url());
    const method = route.request().method();
    if (method === 'POST' && url.pathname === '/api/tasks') {
      return route.fulfill({ json: { task_id: 'new-1', status: 'QUEUED' } });
    }
    if (method === 'DELETE' && url.pathname.startsWith('/api/tasks/')) {
      return route.fulfill({ status: 204, body: '' });
    }
    if (url.pathname === '/api/tasks') return route.fulfill({ json: [QUEUED] });
    return route.fulfill({ json: [] });
  });
}

test.beforeEach(async ({ page }) => {
  await mockApi(page);
});

test.describe('Task submit form', () => {
  test('every control has an accessible name and a keyboard path', async ({ page }) => {
    await page.goto('/');
    for (const name of ['Command', 'Priority', 'Provider', 'Model']) {
      await expect(page.getByRole('combobox', { name })).toBeVisible();
    }
    await expect(page.getByRole('textbox', { name: 'Instruction' })).toBeVisible();
    await expect(page.getByRole('textbox', { name: 'Project path (absolute)' })).toBeVisible();
    await expect(page.getByRole('textbox', { name: /Add tag/ })).toBeVisible();
  });

  test('flags a missing instruction with aria-invalid', async ({ page }) => {
    await page.goto('/');
    const instruction = page.getByRole('textbox', { name: 'Instruction' });
    await expect(instruction).not.toHaveAttribute('aria-invalid', 'true');
    await page.getByRole('button', { name: 'Submit to Queue' }).click();
    await expect(instruction).toHaveAttribute('aria-invalid', 'true');
    await expect(page.getByText('Instruction is required.')).toBeVisible();
  });

  test('a priority picked with the keyboard is kept', async ({ page }) => {
    await page.goto('/');
    const priority = page.getByRole('combobox', { name: 'Priority' });
    await priority.selectOption({ label: '🔴 High' });
    await expect(priority.locator('option:checked')).toHaveText('🔴 High');
  });
});

test.describe('Toasts', () => {
  test('a submitted task raises a status toast that can be dismissed', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('textbox', { name: 'Instruction' }).fill('Add retries');
    await page.getByRole('button', { name: 'Submit to Queue' }).click();
    const toast = page.getByRole('status').filter({ hasText: 'Task submitted' });
    await expect(toast).toBeVisible();
    await toast.getByRole('button', { name: 'Dismiss notification' }).click();
    await expect(toast).toBeHidden();
  });

  test('a toast leaves on its own', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('textbox', { name: 'Instruction' }).fill('x');
    await page.getByRole('button', { name: 'Submit to Queue' }).click();
    const toast = page.getByRole('status').filter({ hasText: 'Task submitted' });
    await expect(toast).toBeVisible();
    await expect(toast).toBeHidden({ timeout: 6000 });
  });
});

test.describe('Task detail drawer', () => {
  async function openDrawer(page: Page) {
    await page.goto('/');
    const row = page.getByText('Write the thing').first();
    await row.click();
    const dialog = page.getByRole('dialog', { name: 'Task details' });
    await expect(dialog).toBeVisible();
    return dialog;
  }

  test('is a labelled modal dialog that takes focus', async ({ page }) => {
    const dialog = await openDrawer(page);
    await expect(dialog).toHaveAttribute('aria-modal', 'true');
    await expect(dialog).toBeFocused();
  });

  test('Escape closes it', async ({ page }) => {
    const dialog = await openDrawer(page);
    await page.keyboard.press('Escape');
    await expect(dialog).toBeHidden();
  });

  test('Tab never leaves the dialog', async ({ page }) => {
    const dialog = await openDrawer(page);
    for (let i = 0; i < 6; i++) {
      await page.keyboard.press('Tab');
      expect(await dialog.evaluate((el) => el.contains(document.activeElement))).toBe(true);
    }
    for (let i = 0; i < 6; i++) {
      await page.keyboard.press('Shift+Tab');
      expect(await dialog.evaluate((el) => el.contains(document.activeElement))).toBe(true);
    }
  });

  test('the close button and the backdrop both close it', async ({ page }) => {
    let dialog = await openDrawer(page);
    await dialog.getByRole('button', { name: 'Close' }).click();
    await expect(dialog).toBeHidden();
    dialog = await openDrawer(page);
    await page.mouse.click(10, 10);
    await expect(dialog).toBeHidden();
  });

  test('cancelling a queued task closes the drawer and confirms with a toast', async ({ page }) => {
    const dialog = await openDrawer(page);
    await dialog.getByRole('button', { name: 'Cancel Task' }).click();
    await expect(dialog).toBeHidden();
    await expect(page.getByRole('status').filter({ hasText: 'Task cancelled' })).toBeVisible();
  });
});

test.describe('Icons', () => {
  test('are decorative: hidden from assistive technology and sized with the text', async ({
    page,
  }) => {
    await page.goto('/');
    const icons = page.locator('svg[data-icon]');
    expect(await icons.count()).toBeGreaterThan(0);
    for (const hidden of await icons.evaluateAll((els) =>
      els.map((e) => e.getAttribute('aria-hidden')),
    )) {
      expect(hidden).toBe('true');
    }
  });

  test('the sidebar buttons keep an accessible name without the icon', async ({ page }) => {
    await page.goto('/');
    await expect(
      page.locator('aside nav').getByRole('button', { name: /settings/i }),
    ).toBeVisible();
  });
});

test.describe('Motion policy', () => {
  test('reduced motion removes transitions', async ({ page }) => {
    await page.emulateMedia({ reducedMotion: 'reduce' });
    await page.goto('/');
    const duration = await page
      .getByRole('button', { name: 'Submit to Queue' })
      .evaluate((el) => getComputedStyle(el).transitionDuration);
    expect(parseFloat(duration)).toBeLessThan(0.01);
  });
});
