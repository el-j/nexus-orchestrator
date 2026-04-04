# TASK-383: Frontend Vitest — view component tests (HistoryView, LiveActivityView, ProvidersView)

**Plan:** PLAN-055 | **Wave:** 5 | **Status:** done | **Role:** testing

## Goal

Create Vitest component tests for the 3 key views fixed in PLAN-054/055. Tests must verify:

- API calls include correct query params (project filtering)
- Status badge filtering works
- Session grouping renders correctly

## Files to Create

- `frontend/src/views/HistoryView.test.ts`
- `frontend/src/views/LiveActivityView.test.ts`

## Required Reading

- `frontend/src/views/HistoryView.vue` — full file
- `frontend/src/views/LiveActivityView.vue` — full file
- `frontend/package.json` — check if `@vue/test-utils` is installed

## Check @vue/test-utils:

```bash
cat frontend/package.json | grep vue-test
```

If not installed: add `@vue/test-utils` and `happy-dom` or use existing `jsdom`:

```bash
cd frontend && npm install --save-dev @vue/test-utils
```

## Test Strategy

### HistoryView.test.ts

```typescript
import { mount } from '@vue/test-utils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HistoryView from './HistoryView.vue';

const mockTasks = [
  {
    id: '1',
    status: 'COMPLETED',
    instruction: 'task 1',
    projectPath: '/p',
    createdAt: '',
    updatedAt: '',
  },
  {
    id: '2',
    status: 'FAILED',
    instruction: 'task 2',
    projectPath: '/p',
    createdAt: '',
    updatedAt: '',
  },
  {
    id: '3',
    status: 'QUEUED',
    instruction: 'task 3',
    projectPath: '/other',
    createdAt: '',
    updatedAt: '',
  },
];

describe('HistoryView', () => {
  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve(mockTasks),
      }),
    );
  });
  afterEach(() => vi.restoreAllMocks());

  it('passes projectPath to API when project is selected', async () => {
    // Mock currentProject to return '/p'
    vi.mock('@/composables/useProjectState', () => ({
      currentProject: { value: '/p' },
    }));
    const wrapper = mount(HistoryView, { global: { stubs: ['RouterLink'] } });
    await wrapper.vm.$nextTick();
    const url = vi.mocked(fetch).mock.calls[0][0] as string;
    expect(url).toContain('projectPath=');
  });

  it('renders status summary bar', async () => {
    const wrapper = mount(HistoryView, { global: { stubs: ['RouterLink'] } });
    await wrapper.vm.$nextTick();
    // Should show COMPLETED, FAILED badges
    expect(wrapper.text()).toContain('Completed');
    expect(wrapper.text()).toContain('Failed');
  });

  it('filters by status badge click', async () => {
    const wrapper = mount(HistoryView, { global: { stubs: ['RouterLink'] } });
    await wrapper.vm.$nextTick();
    // Find completed badge and click
    const completedBadge = wrapper
      .findAll('button')
      .find((btn) => btn.text().includes('Completed'));
    await completedBadge?.trigger('click');
    // Only completed tasks visible
    const taskItems = wrapper.findAll('[data-testid="task-item"]');
    taskItems.forEach((item) => expect(item.text()).toContain('COMPLETED'));
  });
});
```

### LiveActivityView.test.ts

```typescript
describe('LiveActivityView', () => {
  it('renders session groups instead of flat list', async () => {
    const activities = [
      { id: '1', sessionId: 'sess-a', agentName: 'claude', type: 'message', content: 'hello', ... },
      { id: '2', sessionId: 'sess-a', agentName: 'claude', type: 'generation', content: 'response', ... },
      { id: '3', sessionId: 'sess-b', agentName: 'continue', type: 'message', content: 'query', ... },
    ]
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => activities }))
    const wrapper = mount(LiveActivityView, { global: { stubs: ['AIActivityCard'] }})
    await wrapper.vm.$nextTick()
    // Should show 2 session groups (sess-a, sess-b)
    const groups = wrapper.findAll('[data-testid="session-group"]')
    expect(groups).toHaveLength(2)
  })

  it('passes selectedProject to API after project filter change', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => [] }))
    const wrapper = mount(LiveActivityView, { global: { stubs: ['AIActivityCard'] }})
    await wrapper.vm.$nextTick()
    // Change project selection
    const select = wrapper.find('select')
    await select.setValue('/my/project')
    await wrapper.vm.$nextTick()
    // Second fetch call should include projectPath
    const calls = vi.mocked(fetch).mock.calls
    const lastUrl = calls[calls.length - 1][0] as string
    expect(lastUrl).toContain('projectPath=')
  })
})
```

## Note on data-testid

Add `data-testid="session-group"` and `data-testid="task-item"` to the relevant elements in the Vue templates for testability.

## Verification

- `cd frontend && npx vitest run` passes all tests
- At minimum 6 test cases across both view files

## Status

done
