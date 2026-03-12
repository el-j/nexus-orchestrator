import { ref } from 'vue'

// Module-level singleton — shared by useTasks and useProjectFilter without circular deps
export const currentProject = ref<string | null>(
  localStorage.getItem('nexus-project-filter') || null
)

export function setProject(path: string | null): void {
  currentProject.value = path
  if (path) {
    localStorage.setItem('nexus-project-filter', path)
  } else {
    localStorage.removeItem('nexus-project-filter')
  }
}
