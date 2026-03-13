import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { currentProject } from '../composables/useProjectState'
import type { Task } from '../types/domain'
import HistoryView from './HistoryView.vue'

const { getAllTasks } = vi.hoisted(() => ({
  getAllTasks: vi.fn<() => Promise<Task[] | undefined>>(),
}))

vi.mock('../types/wails', () => ({
  getAllTasks,
}))

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://127.0.0.1:9999'),
}))

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: '1234567890abcdef',
    projectPath: '/tmp/project',
    targetFile: 'task.go',
    instruction: 'Task item',
    contextFiles: [],
    modelId: '',
    providerHint: '',
    command: 'auto',
    status: 'COMPLETED',
    createdAt: '2026-03-13T01:00:00.000Z',
    updatedAt: '2026-03-13T01:01:00.000Z',
    logs: '',
    ...overrides,
  }
}

describe('HistoryView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    currentProject.value = null
  })

  it('shows only terminal tasks from getAllTasks', async () => {
    getAllTasks.mockResolvedValue([
      makeTask({ id: 'completed-task', status: 'COMPLETED' }),
      makeTask({ id: 'failed-task', status: 'FAILED' }),
      makeTask({ id: 'queued-task', status: 'QUEUED' }),
    ])

    const wrapper = mount(HistoryView, {
      global: {
        stubs: {
          TaskStatusBadge: {
            props: ['status'],
            template: '<span class="status-badge">{{ status }}</span>',
          },
          TaskDetailDrawer: {
            props: ['task', 'modelValue'],
            template: '<div class="drawer-stub">{{ task?.id ?? "" }}</div>',
          },
        },
      },
    })

    await flushPromises()

    expect(getAllTasks).toHaveBeenCalledTimes(1)
    expect(wrapper.text()).toContain('complete')
    expect(wrapper.text()).toContain('failed-task')
    expect(wrapper.text()).not.toContain('queued-task')
    expect(wrapper.text()).toContain('COMPLETED')
    expect(wrapper.text()).toContain('FAILED')
  })

  it('applies the current project filter and handles undefined task arrays', async () => {
    currentProject.value = '/tmp/project'
    getAllTasks.mockResolvedValue(undefined)

    const wrapper = mount(HistoryView, {
      global: {
        stubs: {
          TaskStatusBadge: true,
          TaskDetailDrawer: true,
        },
      },
    })

    await flushPromises()

    expect(getAllTasks.mock.calls.length).toBeGreaterThanOrEqual(1)
    expect(wrapper.text()).toContain('No task history yet.')
  })
})