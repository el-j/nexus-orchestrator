import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import ProjectBrainCard from './ProjectBrainCard.vue';
import type { BrainStatus, DiscoveredPlanFile } from '../types/domain';

const api = vi.hoisted(() => ({
  getBrainStatus: vi.fn(),
  ingestKnowledge: vi.fn(),
  initProject: vi.fn(),
}));
vi.mock('../types/wails', () => api);

const status = (o: Partial<BrainStatus> = {}): BrainStatus =>
  ({
    initialized: true,
    entryCount: 3,
    totalTokens: 12345,
    lastUpdated: '2026-01-02T03:04:05Z',
    ...o,
  }) as BrainStatus;
const planFile = (o: Partial<DiscoveredPlanFile> = {}): DiscoveredPlanFile =>
  ({
    path: '/p/CLAUDE.md',
    summary: 'Project rules',
    isActive: true,
    lastModified: '2026-01-02T00:00:00Z',
    ...o,
  }) as DiscoveredPlanFile;

async function mountCard(props: { projectPath?: string; nexusFile?: DiscoveredPlanFile } = {}) {
  const w = mount(ProjectBrainCard, { props: { projectPath: '/p', ...props } });
  await flushPromises();
  return w;
}
const button = (w: Awaited<ReturnType<typeof mountCard>>, text: string) =>
  w.findAll('button').find((b) => b.text().includes(text));

beforeEach(() => {
  Object.values(api).forEach((f) => f.mockReset());
  api.getBrainStatus.mockResolvedValue(status());
});

describe('ProjectBrainCard — status', () => {
  it('shows stats of an initialised brain', async () => {
    const w = await mountCard();
    expect(api.getBrainStatus).toHaveBeenCalledWith('/p');
    expect(w.text()).toContain('nexus');
    expect(w.text()).toContain('3');
    expect(w.text()).toContain((12345).toLocaleString());
    expect(w.text()).toContain('Last Sync');
    expect(w.text()).not.toContain('uninitialized');
    expect(button(w, 'Initialize Brain')).toBeUndefined();
  });

  it('omits Last Sync when there is no timestamp', async () => {
    api.getBrainStatus.mockResolvedValue(status({ lastUpdated: '' }));
    expect((await mountCard()).text()).not.toContain('Last Sync');
  });

  it('flags an uninitialised brain and offers to initialise it', async () => {
    api.getBrainStatus.mockResolvedValue(status({ initialized: false, entryCount: 0 }));
    const w = await mountCard();
    expect(w.text()).toContain('uninitialized');
    expect(button(w, 'Initialize Brain')).toBeDefined();
  });

  it('offers to initialise an initialised but empty brain', async () => {
    api.getBrainStatus.mockResolvedValue(status({ entryCount: 0 }));
    expect(button(await mountCard(), 'Initialize Brain')).toBeDefined();
  });

  it('survives a failing status call', async () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    api.getBrainStatus.mockRejectedValue(new Error('down'));
    const w = await mountCard();
    expect(w.text()).toContain('uninitialized');
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it('reloads when the project changes', async () => {
    const w = await mountCard();
    await w.setProps({ projectPath: '/other' });
    expect(api.getBrainStatus).toHaveBeenLastCalledWith('/other');
  });

  it('shows the plan file name, summary, active badge and date', async () => {
    const w = await mountCard({ nexusFile: planFile() });
    expect(w.text()).toContain('CLAUDE.md');
    expect(w.text()).toContain('Project rules');
    expect(w.text()).toContain('active');
  });

  it('omits the summary and active badge when absent', async () => {
    const w = await mountCard({ nexusFile: planFile({ summary: '', isActive: false }) });
    expect(w.text()).not.toContain('Project rules');
    expect(w.text()).not.toContain('active');
  });
});

describe('ProjectBrainCard — initialise', () => {
  beforeEach(() =>
    api.getBrainStatus.mockResolvedValue(status({ initialized: false, entryCount: 0 })),
  );

  it('initialises with the plan file path and shows the result', async () => {
    api.initProject.mockResolvedValue(status());
    const w = await mountCard({ nexusFile: planFile() });
    await button(w, 'Initialize Brain')!.trigger('click');
    await flushPromises();
    expect(api.initProject).toHaveBeenCalledWith('/p', '/p/CLAUDE.md');
    expect(w.text()).toContain('Brain initialized.');
    expect(w.text()).toContain('Entries'); // status replaced
  });

  it('lets the daemon auto-detect when there is no plan file', async () => {
    api.initProject.mockResolvedValue(status());
    const w = await mountCard();
    await button(w, 'Initialize Brain')!.trigger('click');
    await flushPromises();
    expect(api.initProject).toHaveBeenCalledWith('/p', '');
  });

  it('shows an Error message, or the stringified value, and the result can be dismissed', async () => {
    api.initProject.mockRejectedValueOnce(new Error('no CLAUDE.md'));
    const w = await mountCard();
    await button(w, 'Initialize Brain')!.trigger('click');
    await flushPromises();
    expect(w.text()).toContain('no CLAUDE.md');

    api.initProject.mockRejectedValueOnce('plain');
    await button(w, 'Initialize Brain')!.trigger('click');
    await flushPromises();
    expect(w.text()).toContain('plain');

    await w
      .findAll('button')
      .find((b) => b.text() === '✕')!
      .trigger('click');
    expect(w.text()).not.toContain('plain');
  });

  it('disables the button and shows progress while running', async () => {
    let done!: (s: BrainStatus) => void;
    api.initProject.mockReturnValue(new Promise<BrainStatus>((r) => (done = r)));
    const w = await mountCard();
    await button(w, 'Initializing…') // not yet
      ?.trigger('click');
    await button(w, 'Initialize Brain')!.trigger('click');
    expect(button(w, 'Initializing…')!.attributes('disabled')).toBeDefined();
    done(status());
    await flushPromises();
    expect(button(w, 'Initializing…')).toBeUndefined();
  });
});

describe('ProjectBrainCard — sync and ingest', () => {
  it('syncs the plan file, reports the section count and refreshes status', async () => {
    api.ingestKnowledge.mockResolvedValue(7);
    const w = await mountCard({ nexusFile: planFile() });
    api.getBrainStatus.mockClear();
    await button(w, 'Sync Brain')!.trigger('click');
    await flushPromises();
    expect(api.ingestKnowledge).toHaveBeenCalledWith('/p', '/p/CLAUDE.md');
    expect(w.text()).toContain('Ingested 7 sections from CLAUDE.md.');
    expect(api.getBrainStatus).toHaveBeenCalledTimes(1);
  });

  it('reports a failed sync', async () => {
    api.ingestKnowledge.mockRejectedValue(new Error('read failed'));
    const w = await mountCard({ nexusFile: planFile() });
    await button(w, 'Sync Brain')!.trigger('click');
    await flushPromises();
    expect(w.text()).toContain('read failed');
    expect(button(w, 'Sync Brain')).toBeDefined(); // re-enabled
  });

  it('has no Sync button without a plan file, and no Ingest button without a project', async () => {
    expect(button(await mountCard(), 'Sync Brain')).toBeUndefined();
    expect(button(await mountCard({ projectPath: '' }), 'Ingest File')).toBeUndefined();
  });

  it('opens the file picker from Ingest File…', async () => {
    const w = await mountCard();
    const input = w.find('input[type="file"]').element as HTMLInputElement;
    const click = vi.spyOn(input, 'click').mockImplementation(() => {});
    await button(w, 'Ingest File')!.trigger('click');
    expect(click).toHaveBeenCalled();
  });

  async function pick(w: Awaited<ReturnType<typeof mountCard>>, names: string[]) {
    const input = w.find('input[type="file"]');
    Object.defineProperty(input.element, 'files', {
      value: names.map((name) => ({ name })),
      configurable: true,
    });
    await input.trigger('change');
    await flushPromises();
  }

  it('ingests every selected file and totals the sections', async () => {
    api.ingestKnowledge.mockResolvedValueOnce(2).mockResolvedValueOnce(3);
    const w = await mountCard();
    await pick(w, ['a.md', 'b.md']);
    expect(api.ingestKnowledge).toHaveBeenCalledTimes(2);
    expect(w.text()).toContain('Ingested 5 sections from 2 files.');
  });

  it('singularises one file and lists the ones that failed', async () => {
    api.ingestKnowledge.mockResolvedValueOnce(4).mockRejectedValueOnce(new Error('x'));
    const w = await mountCard();
    await pick(w, ['a.md', 'bad.md']);
    expect(w.text()).toContain('Ingested 4 sections from 1 file.');
    expect(w.text()).toContain('Failed: bad.md');
  });

  it('ignores an empty selection', async () => {
    const w = await mountCard();
    await pick(w, []);
    expect(api.ingestKnowledge).not.toHaveBeenCalled();
  });
});
