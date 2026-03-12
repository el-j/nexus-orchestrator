import { computed } from 'vue'
import { useTasks } from './useTasks'
import { currentProject, setProject } from './useProjectState'

export function useProjectFilter() {
  const { tasks } = useTasks()

  const projectList = computed<string[]>(() => {
    const paths = new Set(tasks.value.map(t => t.projectPath).filter(Boolean))
    return Array.from(paths).sort()
  })

  return { currentProject, projectList, setProject }
}
