import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { ref } from 'vue';
import DiscoveredPlansView from './DiscoveredPlansView.vue';
import type { DiscoveredPlanFile } from '../types/domain';

const h = vi.hoisted(() => ({
  scan: vi.fn(),
  createDraft: vi.fn(),
  toastAdd: vi.fn(),
  state: { plans: [] as unknown[], loading: false, error: null as string | null },
}));
vi.mock('../composables/useDiscoveredPlans', () => ({
  useDiscoveredPlans: () => ({
    plans: ref(h.state.plans),
    loading: ref(h.state.loading),
    error: ref(h.state.error),
    scan: h.scan,
  }),
}));
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  return { currentProject: ref(null) };
});
vi.mock('../types/wails', () => ({ createDraft: h.createDraft }));
vi.mock('../composables/useToast', () => ({ useToast: () => ({ add: h.toastAdd }) }));

const Brain = {
  name: 'ProjectBrainCard',
  props: ['projectPath', 'nexusFile'],
  template: '<div class="brain" />',
};

let n = 0;
const plan = (o: Partial<DiscoveredPlanFile> = {}): DiscoveredPlanFile =>
  ({
    id: `f${n++}`,
    projectPath: '/work/app',
    kind: 'markdown',
    path: '/work/app/README.md',
    summary: '',
    isActive: false,
    lastModified: '2026-01-02T00:00:00Z',
    ...o,
  }) as DiscoveredPlanFile;
const task = (num: number, o: Partial<DiscoveredPlanFile> = {}) =>
  plan({
    kind: 'claude-task',
    path: `/work/app/.claude/tasks/TASK-${num}.md`,
    summary: `**Plan:** PLAN-1\n**Status:** todo`,
    ...o,
  });

const mountView = async (plans: DiscoveredPlanFile[] = [], extra: Partial<typeof h.state> = {}) => {
  h.state = { plans, loading: false, error: null, ...extra };
  const w = mount(DiscoveredPlansView, { global: { stubs: { ProjectBrainCard: Brain } } });
  await flushPromises();
  return w;
};
const btn = (w: Awaited<ReturnType<typeof mountView>>, text: string) =>
  w.findAll('button').find((b) => b.text().includes(text))!;

beforeEach(() => {
  n = 0;
  h.scan.mockReset();
  h.createDraft.mockReset().mockResolvedValue('D-1');
  h.toastAdd.mockReset();
});

describe('DiscoveredPlansView — states', () => {
  it('shows the empty state', async () => {
    expect((await mountView()).text()).toContain('No plan files found.');
  });
  it('shows a spinner for the first load, then the error', async () => {
    expect((await mountView([], { loading: true })).find('.animate-spin').exists()).toBe(true);
    expect((await mountView([], { error: 'HTTP 503' })).text()).toContain('HTTP 503');
  });
  it('scans from the header and shows progress', async () => {
    const w = await mountView();
    await btn(w, 'Scan Now').trigger('click');
    expect(h.scan).toHaveBeenCalled();
    const busy = await mountView([plan()], { loading: true });
    expect(busy.text()).toContain('Scanning…');
  });
  it('counts plans and projects with pluralisation', async () => {
    const one = await mountView([plan()]);
    expect(one.find('header').text()).toContain('project');
    expect(one.find('header').text()).not.toContain('projects');
    const two = await mountView([plan(), plan({ projectPath: '/other' })]);
    expect(two.find('header').text()).toContain('projects');
  });
});

describe('DiscoveredPlansView — grouping', () => {
  it('groups task files by parent plan, newest plan and task first, ungrouped last', async () => {
    const w = await mountView([
      task(1),
      task(7, { summary: '**Plan:** PLAN-1\n**Status:** todo' }),
      task(3, { summary: '**Plan:** PLAN-9\n**Status:** done' }),
      task(4, { summary: 'no parent' }),
    ]);
    const t = w.text();
    expect(t.indexOf('PLAN-9')).toBeLessThan(t.indexOf('PLAN-1'));
    expect(t.indexOf('PLAN-1')).toBeLessThan(t.indexOf('Other Tasks'));
    expect(t.indexOf('TASK-7.md')).toBeLessThan(t.indexOf('TASK-1.md'));
    expect(t).toContain('2 tasks');
    expect(t).toContain('1 task');
  });

  it('shows project stats, an active marker, and the brain card with the nexus file', async () => {
    const nexus = plan({ kind: 'nexus', path: '/work/app/NEXUS.md', isActive: true });
    const w = await mountView([
      nexus,
      task(1),
      plan({ kind: 'cursor', path: '/work/app/.cursorrules' }),
    ]);
    expect(w.text()).toContain('1 plans · 1 tasks');
    expect(w.text()).toContain('1 other');
    expect(w.find('section .bg-emerald-400.rounded-full').exists()).toBe(true);
    expect(w.findComponent(Brain).props()).toMatchObject({
      projectPath: '/work/app',
      nexusFile: nexus,
    });
  });

  it('lists other files with their kind badge and shortened summary', async () => {
    const w = await mountView([
      plan({ kind: 'cursor', path: '/p/.cursorrules', summary: 'x'.repeat(120) }),
    ]);
    expect(w.text()).toContain('Other Files');
    expect(w.text()).toContain('cursor');
    expect(w.text()).toContain('x'.repeat(80) + '…');
  });

  it('files plans without a project under "(unknown)"', async () => {
    const w = await mountView([plan({ projectPath: '' })]);
    expect(w.text()).toContain('(unknown)');
  });

  it('colours task status badges and the plan border when work is open', async () => {
    const w = await mountView([
      task(1),
      task(2, { summary: '**Plan:** PLAN-2\nstatus: done' }),
      task(3, { summary: '**Plan:** PLAN-3\n**Status:** weird' }),
    ]);
    expect(w.text()).toContain('todo');
    expect(w.text()).toContain('done');
    expect(w.text()).toContain('weird');
    expect(w.findAll('.border-violet-500\\/50')).toHaveLength(1);
  });
});

describe('DiscoveredPlansView — promote', () => {
  it('offers Promote for todo tasks only and saves a draft with feedback', async () => {
    const w = await mountView([
      task(1),
      task(2, { summary: '**Plan:** PLAN-1\n**Status:** done' }),
    ]);
    const promotes = w.findAll('button').filter((b) => b.text() === 'Promote');
    expect(promotes).toHaveLength(1);
    await promotes[0].trigger('click');
    await flushPromises();
    expect(h.createDraft).toHaveBeenCalledWith(
      expect.objectContaining({
        projectPath: '/work/app',
        targetFile: expect.stringContaining('TASK-1'),
      }),
    );
    expect(h.toastAdd).toHaveBeenCalledWith(
      expect.objectContaining({ severity: 'success', detail: expect.stringContaining('D-1') }),
    );
  });

  it('promotes the task summary as the draft instruction', async () => {
    const w = await mountView([task(1)]);
    await btn(w, 'Promote').trigger('click');
    await flushPromises();
    expect(h.createDraft).toHaveBeenCalledWith(
      expect.objectContaining({ instruction: expect.stringContaining('**Status:** todo') }),
    );
  });

  it('reports a failed promote instead of throwing', async () => {
    h.createDraft.mockRejectedValueOnce(new Error('queue full')).mockRejectedValueOnce('plain');
    const w = await mountView([task(1)]);
    await btn(w, 'Promote').trigger('click');
    await flushPromises();
    expect(h.toastAdd).toHaveBeenLastCalledWith(
      expect.objectContaining({ severity: 'error', detail: 'queue full' }),
    );
    await btn(w, 'Promote').trigger('click');
    await flushPromises();
    expect(h.toastAdd).toHaveBeenLastCalledWith(expect.objectContaining({ detail: 'plain' }));
    expect(btn(w, 'Promote').attributes('disabled')).toBeUndefined();
  });
});

describe('DiscoveredPlansView — filters and collapse', () => {
  const data = () => [
    plan({ kind: 'nexus', path: '/work/app/NEXUS.md', isActive: true }),
    task(1),
    plan({ kind: 'cursor', path: '/work/app/.cursorrules' }),
  ];

  it('hides kind chips when only one kind exists', async () => {
    expect((await mountView([plan()])).text()).not.toContain('Clear filters');
    expect((await mountView([plan()])).findAll('button').some((b) => b.text() === 'Markdown')).toBe(
      false,
    );
  });

  it('filters by kind, combines kinds, and clears', async () => {
    const w = await mountView(data());
    await btn(w, 'Cursor').trigger('click');
    expect(w.text()).toContain('.cursorrules');
    expect(w.text()).not.toContain('TASK-1.md');
    await btn(w, 'Tasks').trigger('click');
    expect(w.text()).toContain('TASK-1.md');
    await btn(w, 'Cursor').trigger('click'); // toggle off
    expect(w.text()).not.toContain('.cursorrules');
    await btn(w, 'Clear filters').trigger('click');
    expect(w.text()).toContain('.cursorrules');
    expect(w.text()).not.toContain('Clear filters');
  });

  it('drops projects with nothing matching the filter', async () => {
    const w = await mountView([
      ...data(),
      plan({ projectPath: '/other', kind: 'cursor', path: '/other/.cursorrules' }),
    ]);
    await btn(w, 'Nexus').trigger('click');
    expect(w.findAll('section')).toHaveLength(1);
  });

  it('orders and labels the kind chips', async () => {
    const w = await mountView([
      plan({ kind: 'markdown' }),
      plan({ kind: 'crewai' }),
      plan({ kind: 'mcp-config' }),
      plan({ kind: 'claude' }),
      ...data(),
    ]);
    const labels = w.findAll('.flex-wrap.mb-4 button').map((b) => b.text());
    expect(labels).toEqual([
      'Nexus',
      'Tasks',
      'Instructions',
      'Cursor',
      'Config',
      'CrewAI',
      'Markdown',
    ]);
  });

  it('auto-collapses projects without active items, and Expand/Collapse All toggles', async () => {
    const w = await mountView([
      plan({ projectPath: '/idle' }),
      plan({ projectPath: '/live', isActive: true }),
    ]);
    const bodies = () => w.findAll('section > div').map((d) => d.attributes('style') ?? '');
    expect(bodies()[0]).toContain('display: none');
    expect(bodies()[1]).not.toContain('display: none');

    await btn(w, 'Collapse All').trigger('click');
    expect(bodies().every((s) => s.includes('display: none'))).toBe(true);
    await btn(w, 'Expand All').trigger('click');
    expect(bodies().some((s) => s.includes('display: none'))).toBe(false);
  });

  it('toggles a single project by clicking its header', async () => {
    const w = await mountView([plan({ isActive: true })]);
    const body = () => w.find('section > div').attributes('style') ?? '';
    await w.find('section > button').trigger('click');
    expect(body()).toContain('display: none');
    await w.find('section > button').trigger('click');
    expect(body()).not.toContain('display: none');
  });
});

describe('DiscoveredPlansView — ordering', () => {
  it('orders task files by number, newest first, and files without a number last', async () => {
    const w = await mountView([
      task(2),
      task(10),
      plan({
        kind: 'claude-task',
        path: '/work/app/.claude/tasks/notes.md',
        summary: '**Plan:** PLAN-1\n**Status:** todo',
      }),
      plan({
        kind: 'claude-task',
        path: '/work/app/.claude/tasks/more.md',
        summary: '**Plan:** PLAN-1\n**Status:** todo',
      }),
    ]);
    const t = w.text();
    expect(t.indexOf('TASK-10')).toBeLessThan(t.indexOf('TASK-2'));
    expect(t.indexOf('TASK-2')).toBeLessThan(t.indexOf('notes.md'));
  });

  it('marks an active task file with a green dot', async () => {
    const w = await mountView([task(1, { isActive: true })]);
    expect(w.find('.bg-emerald-400').exists()).toBe(true);
  });

  describe('DiscoveredPlansView — grouping edge cases', () => {
    it('lists ungrouped tasks after the numbered plans and tolerates a missing summary', async () => {
      const w = await mountView([
        plan({ kind: 'claude-task', path: '/work/app/.claude/tasks/TASK-1.md', summary: '' }),
        task(2),
        task(3, { summary: '**Plan:** PLAN-7\n**Status:** todo' }),
      ]);
      const t = w.text();
      expect(t.indexOf('PLAN-7')).toBeLessThan(t.indexOf('PLAN-1'));
      expect(t).toContain('TASK-1');
    });
  });
});
