import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { ref } from 'vue';
import BrainView from './BrainView.vue';

interface Fake {
  path: string;
  status: ReturnType<typeof ref>;
  entries: ReturnType<typeof ref>;
  fileMap: ReturnType<typeof ref>;
  searchResults: ReturnType<typeof ref>;
  loading: ReturnType<typeof ref>;
  error: ReturnType<typeof ref>;
  fetchStatus: ReturnType<typeof vi.fn>;
  init: ReturnType<typeof vi.fn>;
  ingest: ReturnType<typeof vi.fn>;
  listEntries: ReturnType<typeof vi.fn>;
  deleteEntry: ReturnType<typeof vi.fn>;
  search: ReturnType<typeof vi.fn>;
  fetchFileMap: ReturnType<typeof vi.fn>;
}

const h = vi.hoisted(() => ({
  instances: [] as unknown[],
  project: { value: null as string | null },
}));

vi.mock('../composables/useBrain', async () => {
  const { ref } = await import('vue');
  return {
    useBrain: (path: string) => {
      const fake = {
        path,
        status: ref(null),
        entries: ref([]),
        fileMap: ref([]),
        searchResults: ref([]),
        loading: ref(false),
        error: ref(null),
        fetchStatus: vi.fn().mockResolvedValue(undefined),
        init: vi.fn().mockResolvedValue(undefined),
        ingest: vi.fn().mockResolvedValue(2),
        listEntries: vi.fn().mockResolvedValue(undefined),
        deleteEntry: vi.fn().mockResolvedValue(undefined),
        search: vi.fn().mockResolvedValue(undefined),
        fetchFileMap: vi.fn().mockResolvedValue(undefined),
      };
      h.instances.push(fake);
      return fake;
    },
  };
});
vi.mock('../composables/useProjectState', async () => {
  const { ref } = await import('vue');
  const currentProject = ref<string | null>(null);
  h.project = currentProject as unknown as { value: string | null };
  return { currentProject };
});

const inst = () => h.instances as Fake[];
const last = () => inst()[inst().length - 1];
const mountView = async () => {
  const w = mount(BrainView);
  await flushPromises();
  return w;
};
const tab = (w: Awaited<ReturnType<typeof mountView>>, label: string) =>
  w.findAll('button').find((b) => b.text() === label)!;
const pathInput = (w: Awaited<ReturnType<typeof mountView>>) =>
  w.find('input[placeholder="Project path…"]');

beforeEach(() => {
  h.instances.length = 0;
  h.project.value = null;
  localStorage.clear();
});

describe('BrainView — project selection', () => {
  it('starts without a project and does not load anything', async () => {
    await mountView();
    expect(inst()).toHaveLength(1);
    expect(last().path).toBe('');
    expect(last().fetchStatus).not.toHaveBeenCalled();
  });

  it('restores the last project from localStorage and loads its status', async () => {
    localStorage.setItem('nexus:projectPath', '/saved');
    const w = await mountView();
    expect((pathInput(w).element as HTMLInputElement).value).toBe('/saved');
    expect(last().path).toBe('/saved');
    expect(last().fetchStatus).toHaveBeenCalledTimes(1);
  });

  it('falls back to the sidebar project when nothing was saved', async () => {
    h.project.value = '/sidebar';
    const w = await mountView();
    expect((pathInput(w).element as HTMLInputElement).value).toBe('/sidebar');
    expect(last().path).toBe('/sidebar');
  });

  it('switches project on Enter, persists it and loads status', async () => {
    const w = await mountView();
    await pathInput(w).setValue('/new');
    await pathInput(w).trigger('keydown.enter');
    expect(localStorage.getItem('nexus:projectPath')).toBe('/new');
    expect(last().path).toBe('/new');
    expect(last().fetchStatus).toHaveBeenCalled();
  });

  it('switches project on blur, and ignores an empty path', async () => {
    const w = await mountView();
    await pathInput(w).trigger('blur');
    expect(inst()).toHaveLength(1);
    await pathInput(w).setValue('/blurred');
    await pathInput(w).trigger('blur');
    expect(last().path).toBe('/blurred');
  });

  it('renders the NEW project’s state after switching (regression: shallowRef)', async () => {
    const w = await mountView();
    await pathInput(w).setValue('/p');
    await pathInput(w).trigger('blur');
    (last().status as { value: unknown }).value = {
      initialized: true,
      entryCount: 7,
      totalTokens: 12345,
      kindCounts: {},
    };
    await flushPromises();
    expect(w.find('header').text()).toContain('7');
    expect(w.find('header').text()).toContain((12345).toLocaleString());
  });
});

describe('BrainView — header', () => {
  it('shows zero counts without a status', async () => {
    const t = (await mountView()).find('header').text();
    expect(t).toContain('0 entries');
    expect(t).toContain('0 tokens');
  });

  it('offers Initialize only for an uninitialised brain with a path', async () => {
    const w = await mountView();
    const init = () => w.findAll('button').find((b) => b.text() === 'Initialize')!;
    expect(init().attributes('disabled')).toBeDefined(); // no path yet
    await pathInput(w).setValue('/p');
    await pathInput(w).trigger('blur');
    await flushPromises();
    expect(init().attributes('disabled')).toBeUndefined();
    await init().trigger('click');
    expect(last().init).toHaveBeenCalled();

    (last().status as { value: unknown }).value = { initialized: true, kindCounts: {} };
    await flushPromises();
    expect(w.findAll('button').some((b) => b.text() === 'Initialize')).toBe(false);
  });

  it('refreshes only when a project is set', async () => {
    const w = await mountView();
    await tab(w, 'Refresh').trigger('click');
    expect(last().fetchStatus).not.toHaveBeenCalled();
    await pathInput(w).setValue('/p');
    await pathInput(w).trigger('blur');
    const calls = last().fetchStatus.mock.calls.length;
    await tab(w, 'Refresh').trigger('click');
    expect(last().fetchStatus.mock.calls.length).toBe(calls + 1);
  });

  it('shows errors and the per-kind counts', async () => {
    const w = await mountView();
    (last().error as { value: unknown }).value = 'brain offline';
    (last().status as { value: unknown }).value = {
      initialized: true,
      kindCounts: { architecture: 3, glossary: 1 },
    };
    await flushPromises();
    expect(w.text()).toContain('brain offline');
    expect(w.text()).toContain('architecture');
    expect(w.text()).toContain('glossary');
  });

  it('disables actions while loading', async () => {
    const w = await mountView();
    (last().loading as { value: boolean }).value = true;
    await flushPromises();
    expect(tab(w, 'Refresh').attributes('disabled')).toBeDefined();
  });
});

describe('BrainView — search tab', () => {
  it('does nothing for an empty query and disables the button', async () => {
    const w = await mountView();
    expect(tab(w, 'Search').attributes('disabled')).toBeUndefined(); // tab button
    const go = w.findAll('button').filter((b) => b.text() === 'Search')[1];
    expect(go.attributes('disabled')).toBeDefined();
    await w.find('input[placeholder="Search knowledge…"]').trigger('keydown.enter');
    expect(last().search).not.toHaveBeenCalled();
  });

  it('searches on click and on Enter', async () => {
    const w = await mountView();
    const input = w.find('input[placeholder="Search knowledge…"]');
    await input.setValue('auth');
    await w
      .findAll('button')
      .filter((b) => b.text() === 'Search')[1]
      .trigger('click');
    await input.trigger('keydown.enter');
    expect(last().search).toHaveBeenCalledTimes(2);
    expect(last().search).toHaveBeenCalledWith('auth');
  });

  it('shows results, or an empty hint', async () => {
    const w = await mountView();
    expect(w.text()).toContain('No results');
    (last().searchResults as { value: unknown }).value = [
      { kind: 'learning', topic: 'Retries', content: 'use backoff', tokens: 12 },
    ];
    await flushPromises();
    expect(w.text()).toContain('Retries');
    expect(w.text()).toContain('use backoff');
    expect(w.text()).toContain('12t');
    expect(w.text()).not.toContain('No results');
  });
});

describe('BrainView — entries tab', () => {
  async function open() {
    const w = await mountView();
    await tab(w, 'Entries').trigger('click');
    return w;
  }

  it('shows an empty hint, then the entries', async () => {
    const w = await open();
    expect(w.text()).toContain('No entries');
    (last().entries as { value: unknown }).value = [
      { id: 'e1', kind: 'convention', topic: 'Naming', content: 'camelCase', tokenCount: 4 },
    ];
    await flushPromises();
    expect(w.text()).toContain('Naming');
    expect(w.text()).toContain('4t');
  });

  it('filters by kind', async () => {
    const w = await open();
    await w.find('select').setValue('glossary');
    expect(last().listEntries).toHaveBeenCalledWith('glossary');
  });

  it('deletes an entry', async () => {
    const w = await open();
    (last().entries as { value: unknown }).value = [
      { id: 'e1', kind: 'k', topic: 't', content: 'c', tokenCount: 1 },
    ];
    await flushPromises();
    await w.find('button[title="Delete"]').trigger('click');
    expect(last().deleteEntry).toHaveBeenCalledWith('e1');
  });
});

describe('BrainView — file map tab', () => {
  it('loads and numbers the file map', async () => {
    const w = await mountView();
    await tab(w, 'File Map').trigger('click');
    expect(w.text()).toContain('No file map entries');
    await tab(w, 'Load File Map').trigger('click');
    expect(last().fetchFileMap).toHaveBeenCalled();
    (last().fileMap as { value: unknown }).value = ['a.go', 'b.go'];
    await flushPromises();
    expect(w.text()).toContain('1. a.go');
    expect(w.text()).toContain('2. b.go');
  });
});

describe('BrainView — ingest tab', () => {
  async function open(project = '/p') {
    const w = await mountView();
    await pathInput(w).setValue(project);
    await pathInput(w).trigger('blur');
    await tab(w, 'Ingest').trigger('click');
    return w;
  }
  const pick = async (w: Awaited<ReturnType<typeof mountView>>, names: string[]) => {
    const input = w.find('input[type="file"]');
    Object.defineProperty(input.element, 'files', {
      value: names.map((name) => ({ name })),
      configurable: true,
    });
    await input.trigger('change');
    await flushPromises();
  };

  it('requires a project before files can be chosen', async () => {
    const w = await mountView();
    await tab(w, 'Ingest').trigger('click');
    expect(
      w
        .findAll('button')
        .find((b) => b.text() === 'Select Files…')!
        .attributes('disabled'),
    ).toBeDefined();
  });

  it('opens the file picker', async () => {
    const w = await open();
    const input = w.find('input[type="file"]').element as HTMLInputElement;
    const click = vi.spyOn(input, 'click').mockImplementation(() => {});
    await tab(w, 'Select Files…').trigger('click');
    expect(click).toHaveBeenCalled();
  });

  it('ingests every file and reports the total', async () => {
    const w = await open();
    await pick(w, ['a.md', 'b.md']);
    expect(last().ingest).toHaveBeenCalledTimes(2);
    expect(w.text()).toContain('Ingested 4 sections from 2 files.');
  });

  it('singularises a single file', async () => {
    const w = await open();
    await pick(w, ['a.md']);
    expect(w.text()).toContain('from 1 file.');
  });

  it('ignores a selection without a project', async () => {
    const w = await mountView();
    await tab(w, 'Ingest').trigger('click');
    await pick(w, ['a.md']);
    expect(last().ingest).not.toHaveBeenCalled();
  });
});
