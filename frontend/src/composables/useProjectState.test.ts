import { beforeEach, describe, expect, it, vi } from 'vitest';

describe('useProjectState', () => {
  beforeEach(() => {
    vi.resetModules();
    localStorage.clear();
  });

  it('persists the selected project and clears it cleanly', async () => {
    const { currentProject, setProject } = await import('./useProjectState');

    expect(currentProject.value).toBeNull();

    setProject('/repo/a');
    expect(currentProject.value).toBe('/repo/a');
    expect(localStorage.getItem('nexus-project-filter')).toBe('/repo/a');

    setProject(null);
    expect(currentProject.value).toBeNull();
    expect(localStorage.getItem('nexus-project-filter')).toBeNull();
  });
});
