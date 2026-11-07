import { test, expect, type APIRequestContext } from '@playwright/test';
import { mkdtempSync, readFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const API = 'http://127.0.0.1:63987';

async function waitForStatus(request: APIRequestContext, id: string, want: string) {
  await expect
    .poll(async () => (await (await request.get(`${API}/api/tasks/${id}`)).json()).status, {
      timeout: 45_000,
    })
    .toBe(want);
}

test.describe('views against a real daemon', () => {
  test('a discovered command-line client is promoted in the Providers view and then runs a task', async ({
    page,
    request,
  }) => {
    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /providers/i })
      .click();
    await expect(page.getByRole('heading', { name: 'Providers', exact: true })).toBeVisible();

    await page.getByRole('button', { name: /scan now/i }).click();
    const card = page
      .getByText('Google Antigravity CLI')
      .locator('xpath=ancestor::div[.//button[contains(., "Promote to Active")]][1]');
    await expect(card).toBeVisible();
    await card.getByRole('button', { name: /promote to active/i }).click();

    // The form is pre-filled with what the scan found; saving makes it a provider.
    await expect(page.getByRole('heading', { name: /add provider|edit provider/i })).toBeVisible();
    await page.getByRole('button', { name: /^save$/i }).click();
    await expect
      .poll(async () => {
        const providers = (await (await request.get(`${API}/api/providers`)).json()) as {
          name: string;
        }[];
        return providers.some((p) => p.name === 'Google Antigravity CLI');
      })
      .toBe(true);

    const project = mkdtempSync(join(tmpdir(), 'pw-agy-'));
    const res = await request.post(`${API}/api/tasks`, {
      data: {
        projectPath: project,
        targetFile: 'from_agy.go',
        instruction: 'write FUNC_FromAgy',
        providerName: 'Google Antigravity CLI',
      },
    });
    expect(res.status()).toBe(201);
    const { task_id: id } = (await res.json()) as { task_id: string };
    await waitForStatus(request, id, 'COMPLETED');
    expect(readFileSync(join(project, 'from_agy.go'), 'utf8')).toContain('func FromAgy()');
  });

  test('a finished task is listed in History with its status', async ({ page, request }) => {
    const project = mkdtempSync(join(tmpdir(), 'pw-history-'));
    const res = await request.post(`${API}/api/tasks`, {
      data: { projectPath: project, targetFile: 'hist.go', instruction: 'write FUNC_InHistory' },
    });
    const { task_id: id } = (await res.json()) as { task_id: string };
    await waitForStatus(request, id, 'COMPLETED');

    await page.goto('/projects/history');
    await expect(page.getByRole('heading', { name: 'Task History' })).toBeVisible();
    await expect(page.getByText('write FUNC_InHistory').first()).toBeVisible();
    await expect(page.getByRole('button', { name: /completed/i }).first()).toBeVisible();
  });

  test('Live Activity shows its heading and an honest empty state without agent sessions', async ({
    page,
  }) => {
    await page.goto('/projects/live-activity');
    await expect(page.getByRole('heading', { name: 'Live Activity' })).toBeVisible();
    await expect(page.getByText(/no ai activity detected/i)).toBeVisible();
  });

  test('every view shows its own heading', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await page.goto('/');
    const nav = page.locator('aside nav');
    const views: [RegExp, string][] = [
      [/mission control/i, 'Mission Control'],
      [/agents/i, 'Agents'],
      [/providers/i, 'Providers'],
      [/settings/i, 'Settings'],
    ];
    for (const [button, heading] of views) {
      await nav.getByRole('button', { name: button }).click();
      await expect(page.getByRole('heading', { name: heading, exact: true })).toBeVisible();
    }
    await nav.getByRole('button', { name: /projects/i }).click();
    for (const tab of ['plans', 'history', 'backlog', 'brain', 'activity']) {
      await page.getByTestId(`project-tab-${tab}`).click();
      await expect(page.getByTestId(`project-tab-${tab}`)).toHaveAttribute('aria-current', 'page');
    }
    expect(errors).toEqual([]);
  });
});
