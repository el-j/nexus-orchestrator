# TASK-382: Frontend Vitest — composable unit tests

**Plan:** PLAN-055 | **Wave:** 5 | **Status:** done | **Role:** testing

## Goal

Create Vitest unit tests for the 4 core composables. The vitest infrastructure already exists (`vitest.config.ts`, `src/test/setup.ts`, `jsdom` env) but has ZERO test files.

## Files to Create

- `frontend/src/composables/useActivities.test.ts`
- `frontend/src/composables/useProjectState.test.ts`
- `frontend/src/composables/useDiscoveredPlans.test.ts`

## Required Reading Before Implementing

- `frontend/src/composables/useActivities.ts` — full file, understand fetch + SSE
- `frontend/src/composables/useProjectState.ts` — full file, understand localStorage
- `frontend/src/composables/useDiscoveredPlans.ts` — full file, understand projectPath param
- `frontend/src/test/setup.ts` — understand MockEventSource already available

## Test Strategy

### useActivities.test.ts

```typescript
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { nextTick } from 'vue'
import { useActivities } from './useActivities'

describe('useActivities', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn())
  })
  afterEach(() => vi.restoreAllMocks())

  it('fetches activities on mount', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({
      ok: true, json: () => Promise.resolve([{ id: '1', agentName: 'claude', ... }])
    } as Response)
    const { activities, loading } = useActivities()
    expect(loading.value).toBe(true)
    await nextTick()
    expect(activities.value).toHaveLength(1)
    expect(loading.value).toBe(false)
  })

  it('passes projectFilter as ?projectPath= query param', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) } as Response)
    useActivities({ projectFilter: '/my/project' })
    await nextTick()
    const url = vi.mocked(fetch).mock.calls[0][0] as string
    expect(url).toContain('projectPath=%2Fmy%2Fproject')
  })

  it('does not include projectPath when filter is empty', async () => {
    vi.mocked(fetch).mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) } as Response)
    useActivities()
    await nextTick()
    const url = vi.mocked(fetch).mock.calls[0][0] as string
    expect(url).not.toContain('projectPath')
  })

  it('sets error on fetch failure', async () => {
    vi.mocked(fetch).mockRejectedValueOnce(new Error('Network error'))
    const { error, loading } = useActivities()
    await nextTick()
    expect(error.value).toBeTruthy()
    expect(loading.value).toBe(false)
  })
})
```

### useProjectState.test.ts

```typescript
describe('useProjectState', () => {
  beforeEach(() => localStorage.clear());

  it('starts with null when localStorage is empty', () => {
    const { currentProject } = useProjectState();
    expect(currentProject.value).toBeNull();
  });

  it('persists to localStorage on setProject', () => {
    const { setProject, currentProject } = useProjectState();
    setProject('/my/path');
    expect(currentProject.value).toBe('/my/path');
    expect(localStorage.getItem('nexus-project-filter')).toBe('/my/path');
  });

  it('removes from localStorage on setProject(null)', () => {
    localStorage.setItem('nexus-project-filter', '/old');
    const { setProject } = useProjectState();
    setProject(null);
    expect(localStorage.getItem('nexus-project-filter')).toBeNull();
  });
});
```

### useDiscoveredPlans.test.ts

```typescript
describe('useDiscoveredPlans', () => {
  it('includes projectPath in query when provided', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => [] }));
    useDiscoveredPlans('/my/path');
    await nextTick();
    const url = vi.mocked(fetch).mock.calls[0][0] as string;
    expect(url).toContain('projectPath=');
    expect(url).toContain(encodeURIComponent('/my/path'));
  });

  it('omits projectPath when not provided', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => [] }));
    useDiscoveredPlans();
    await nextTick();
    const url = vi.mocked(fetch).mock.calls[0][0] as string;
    expect(url).not.toContain('projectPath');
  });
});
```

## Update vitest.config.ts coverage section:

```typescript
coverage: {
  include: [
    'src/composables/**',
    'src/views/BacklogView.vue',
    'src/views/HistoryView.vue',
    'src/views/LiveActivityView.vue',
  ],
}
```

## Verification

- `cd frontend && npx vitest run` passes all tests
- Coverage report shows composables covered

## Status

done
