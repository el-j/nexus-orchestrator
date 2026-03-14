import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { currentProject } from '../composables/useProjectState'
import type { Task } from '../types/domain'
import BacklogView from './BacklogView.vue'

const { getBacklog } = vi.hoisted(() => ({
  getBacklog: vi.fn<() => Promise<Task[] | undefined>>(),
}))

vi.mock('../types/wails', () => ({
  getBacklog,
}))

vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: vi.fn().mockResolvedValue('http://127.0.0.1:63987'),
}))

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 'draft-1',
    projectPath: '/tmp/project',
    targetFile: 'draft.go',
    instruction: 'Draft item',
    contextFiles: [],
    modelId: '',
    providerHint: '',
    command: 'auto',
    status: 'DRAFT',
    createdAt: '2026-03-13T01:00:00.000Z',
    updatedAt: '2026-03-13T01:00:00.000Z',
    logs: '',
    ...overrides,
  }
}

describe('BacklogView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    currentProject.value = null
  })

  it('loads backlog tasks from getBacklog and refreshes on promote', async () => {
    getBacklog.mockResolvedValue([makeTask()])

    const wrapper = mount(BacklogView, {
      global: {
        stubs: {
          BacklogList: {
            name: 'BacklogList',
            props: ['items'],
            template: '<button class="backlog-stub" @click="$emit(\'promoted\', items[0]?.id)">{{ items.length }}</button>',
          },
        },
      },
    })

    await flushPromises()

    expect(getBacklog).toHaveBeenCalledWith('')
    expect(wrapper.text()).toContain('1')

    await wrapper.get('.backlog-stub').trigger('click')
    await flushPromises()

    expect(getBacklog).toHaveBeenCalledTimes(2)
  })

  it('uses the current project filter and handles undefined backlog arrays', async () => {
    currentProject.value = '/tmp/project'
    getBacklog.mockResolvedValue(undefined)

    const wrapper = mount(BacklogView, {
      global: {
        stubs: {
          BacklogList: {
            name: 'BacklogList',
            props: ['items'],
            template: '<div class="backlog-stub">{{ items.length }}</div>',
          },
        },
      },
    })

    await flushPromises()

    expect(getBacklog).toHaveBeenCalledWith('/tmp/project')
    expect(wrapper.text()).toContain('0')
  })
})