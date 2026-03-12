import { ref, computed } from 'vue'
import { useTasks } from './useTasks'

// Module-level singleton — also imported directly by useTasks to avoid circular call
export const currentProject = ref<string | null>(
  localStorage.getItem('nexus-project-filter') || null
)

export function useProjectFilter() {
  const { tasks } = useTasks()

  const projectList = computed<string[]>(() => {
    const paths = new Set(tasks.value.map(t => t.ProjectPath).filter(Boolean))
    return Array.from(paths).sort()
  })

  function setProject(path: string | null) {
    currentProject.value = path
    if (path) {
      localStorage.setItem('nexus-project-filter', path)
    } else {
      localStorage.removeItem('nexus-project-filter')
    }
  }

  return { currentProject, projectList, setProject }
}
