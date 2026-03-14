<template>
  <div class="rounded-xl border border-white/10 bg-[#0a0a10] overflow-hidden shadow-2xl shadow-violet-500/10">
    <!-- Terminal header -->
    <div class="flex items-center gap-2 px-4 py-3 bg-[#111118] border-b border-white/5">
      <span class="w-3 h-3 rounded-full bg-red-500/70"></span>
      <span class="w-3 h-3 rounded-full bg-yellow-500/70"></span>
      <span class="w-3 h-3 rounded-full bg-green-500/70"></span>
      <span class="ml-3 text-xs text-slate-500 font-mono">nexus-daemon</span>
    </div>
    <!-- Terminal body -->
    <div class="p-5 font-mono text-sm leading-relaxed min-h-[280px]" aria-live="polite">
      <div
        v-for="(line, i) in displayedLines"
        :key="i"
        :class="lineClass(line)"
      >
        {{ line.text }}<span v-if="i === displayedLines.length - 1 && !done" class="terminal-cursor">▋</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

interface TermLine {
  text: string
  type: 'cmd' | 'output' | 'success' | 'info' | 'blank'
}

const lines: TermLine[] = [
  { text: '$ nexus-daemon', type: 'cmd' },
  { text: '  HTTP API   → :63987', type: 'output' },
  { text: '  MCP Server → :63988', type: 'output' },
  { text: '  Dashboard  → :63987/ui', type: 'output' },
  { text: '', type: 'blank' },
  { text: "$ curl -X POST localhost:63987/api/tasks \\", type: 'cmd' },
  { text: "    -d '{\"instruction\":\"Add auth middleware\"}'", type: 'output' },
  { text: '', type: 'blank' },
  { text: '  {"id":"a1b2c3","status":"QUEUED"}', type: 'info' },
  { text: '', type: 'blank' },
  { text: '  Task routed → LM Studio (codellama)', type: 'output' },
  { text: '  Completed in 4.2s ✓', type: 'success' },
]

const displayedLines = ref<TermLine[]>([])
const done = ref(false)
let timeout: ReturnType<typeof setTimeout>

function lineClass(line: TermLine) {
  if (line.type === 'cmd') return 'text-violet-400'
  if (line.type === 'success') return 'text-emerald-400'
  if (line.type === 'info') return 'text-cyan-400'
  if (line.type === 'blank') return 'h-3'
  return 'text-slate-400'
}

function playAnimation() {
  displayedLines.value = []
  done.value = false
  let i = 0
  function next() {
    if (i >= lines.length) {
      done.value = true
      timeout = setTimeout(playAnimation, 4000)
      return
    }
    displayedLines.value.push(lines[i])
    i++
    timeout = setTimeout(next, i === 1 ? 400 : 200)
  }
  next()
}

onMounted(() => {
  timeout = setTimeout(playAnimation, 800)
})
onUnmounted(() => clearTimeout(timeout))
</script>
