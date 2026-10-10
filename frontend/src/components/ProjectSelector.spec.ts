import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import ProjectSelector from './ProjectSelector.vue';

const { currentProject, projectList, setProject } = vi.hoisted(() => ({
  currentProject: { value: null as string | null },
  projectList: { value: [] as string[] },
  setProject: vi.fn(),
}));

vi.mock('../composables/useProjectFilter', () => ({
  useProjectFilter: () => ({
    currentProject: ref(currentProject.value),
    projectList: ref(projectList.value),
    setProject,
  }),
}));

function mountSelector(current: string | null, projects: string[]) {
  currentProject.value = current;
  projectList.value = projects;
  setProject.mockClear();
  return mount(ProjectSelector);
}

describe('ProjectSelector', () => {
  it('lists All Projects plus each project by its folder name', () => {
    const w = mountSelector(null, ['/work/alpha', '/work/beta/']);
    const options = w.findAll('option').map((o) => o.text());
    expect(options).toEqual(['All Projects', 'alpha', 'beta']);
    expect(w.find('option[value="/work/alpha"]').attributes('title')).toBe('/work/alpha');
  });

  it('reflects the current project in the select and its tooltip', () => {
    const w = mountSelector('/work/alpha', ['/work/alpha']);
    const select = w.find('select');
    expect((select.element as HTMLSelectElement).value).toBe('/work/alpha');
    expect(select.attributes('title')).toBe('/work/alpha');
    expect(w.find('.pi-filter').attributes('title')).toBe('alpha');
  });

  it('shows "All Projects" as the tooltip when nothing is selected', () => {
    const w = mountSelector(null, []);
    expect(w.find('select').attributes('title')).toBe('All Projects');
    expect(w.find('.pi-filter').attributes('title')).toBe('All Projects');
  });

  it('sets the chosen project, and null for "All Projects"', async () => {
    const w = mountSelector(null, ['/work/alpha']);
    await w.find('select').setValue('/work/alpha');
    expect(setProject).toHaveBeenLastCalledWith('/work/alpha');
    await w.find('select').setValue('');
    expect(setProject).toHaveBeenLastCalledWith(null);
  });

  it('falls back to the raw path when it has no usable segments', () => {
    const w = mountSelector(null, ['/']);
    expect(w.findAll('option')[1].text()).toBe('/');
  });
});
