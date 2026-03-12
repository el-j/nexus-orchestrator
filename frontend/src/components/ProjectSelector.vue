<template>
  <div class="px-3 py-2 border-b border-white/5">
    <label class="text-xs text-slate-500 uppercase tracking-wider mb-1 hidden lg:block">Project</label>
    <select
      :value="currentProject ?? ''"
      @change="setProject(($event.target as HTMLSelectElement).value || null)"
      class="w-full rounded-lg bg-white/[0.05] text-slate-300 text-xs px-2 py-1.5 border border-white/10
             focus:outline-none focus:border-violet-500/50 hover:border-white/20 transition-colors
             hidden lg:block"
      :title="currentProject ?? 'All Projects'"
    >
      <option value="" class="bg-[#0d0d14]">All Projects</option>
      <option v-for="p in projectList" :key="p" :value="p" :title="p" class="bg-[#0d0d14]">
        {{ baseName(p) }}
      </option>
    </select>
    <!-- Collapsed icon for narrow sidebar -->
    <div class="lg:hidden flex justify-center">
      <i class="pi pi-filter text-slate-500 text-sm" :title="currentProject ? baseName(currentProject) : 'All Projects'"></i>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useProjectFilter } from '../composables/useProjectFilter'

const { currentProject, projectList, setProject } = useProjectFilter()

function baseName(path: string): string {
  return path.split('/').filter(Boolean).pop() ?? path
}
</script>
