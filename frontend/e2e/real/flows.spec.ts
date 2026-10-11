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

test.describe('against a real daemon', () => {
  test('the fake provider is discovered and shown as active', async ({ page, request }) => {
    await expect
      .poll(async () => {
        const providers = (await (await request.get(`${API}/api/providers`)).json()) as {
          name: string;
          active: boolean;
        }[];
        return providers.some((p) => /ollama/i.test(p.name) && p.active);
      })
      .toBe(true);

    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /providers/i })
      .click();
    await expect(page.getByText(/ollama/i).first()).toBeVisible();
  });

  test('a task submitted through the UI runs to completion and writes its file', async ({
    page,
    request,
  }) => {
    const project = mkdtempSync(join(tmpdir(), 'pw-project-'));
    await page.goto('/');

    await page.getByPlaceholder(/what should the ai do/i).fill('write FUNC_FromUi');
    await page.getByPlaceholder(/project path/i).fill(project);
    await page.getByPlaceholder(/target file/i).fill('from_ui.go');
    await page.getByRole('button', { name: /submit to queue/i }).click();

    await expect
      .poll(
        async () => {
          const tasks = (await (await request.get(`${API}/api/tasks/all`)).json()) as {
            projectPath: string;
            status: string;
          }[];
          return tasks.find((t) => t.projectPath === project)?.status;
        },
        { timeout: 45_000 },
      )
      .toBe('COMPLETED');
    expect(readFileSync(join(project, 'from_ui.go'), 'utf8')).toContain('func FromUi()');
  });

  test('a draft shows up in the backlog and can be promoted', async ({ page, request }) => {
    const project = mkdtempSync(join(tmpdir(), 'pw-backlog-'));
    const res = await request.post(`${API}/api/tasks/draft`, {
      data: { projectPath: project, instruction: 'write FUNC_Promoted', targetFile: 'promoted.go' },
    });
    expect(res.status()).toBe(201);
    const { id } = (await res.json()) as { id: string };

    await page.goto('/');
    await page
      .locator('aside nav')
      .getByRole('button', { name: /backlog/i })
      .click();
    await expect(page.getByRole('heading', { name: /^backlog$/i })).toBeVisible();

    const promote = await request.post(`${API}/api/tasks/${id}/promote`);
    expect(promote.ok()).toBe(true);
    await waitForStatus(request, id, 'COMPLETED');
    expect(readFileSync(join(project, 'promoted.go'), 'utf8')).toContain('func Promoted()');
  });

  test('every view renders against live data without script errors', async ({ page }) => {
    const errors: string[] = [];
    page.on('pageerror', (e) => errors.push(e.message));

    await page.goto('/');
    const nav = page.locator('aside nav');
    const labels = [
      'Mission Control',
      'Agents',
      'Providers',
      'Projects',
      'Plans',
      'History',
      'Backlog',
      'Brain',
      'Settings',
    ];
    for (const label of labels) {
      const item = nav.getByRole('button', { name: new RegExp(label, 'i') });
      await item.click();
      await expect(item).toBeVisible();
      await page.waitForTimeout(400); // views poll and stream, so the network never goes idle
    }
    expect(errors).toEqual([]);
  });

  test('knowledge ingested over the API is searchable', async ({ request }) => {
    const project = mkdtempSync(join(tmpdir(), 'pw-brain-'));
    const ingest = await request.post(`${API}/api/brain/ingest-content`, {
      data: {
        projectPath: project,
        name: 'NOTES.md',
        content: '## Architecture\nThe zebra scheduler owns retries.\n',
      },
    });
    expect(ingest.ok()).toBe(true);

    const res = await request.get(
      `${API}/api/brain/search?projectPath=${encodeURIComponent(project)}&q=zebra`,
    );
    expect(res.ok()).toBe(true);
    const { results } = (await res.json()) as { results: unknown[] };
    expect(results.length).toBeGreaterThan(0);
  });
});
