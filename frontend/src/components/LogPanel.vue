<template>
  <div class="log-panel" :style="collapsed ? {} : { height: panelHeight + 'px' }">
    <!-- Header bar -->
    <div
      class="log-header"
      @mousedown="onDragStart"
    >
      <div class="log-header-left">
        <span class="log-title">Logs</span>
        <span :class="['status-dot', connected ? 'connected' : 'disconnected']" :title="connected ? 'Connected' : 'Disconnected'"></span>
        <span v-if="!collapsed" class="log-count">({{ logs.length }})</span>
      </div>
      <div class="log-header-right">
        <select v-if="!collapsed" v-model="levelFilter" class="level-filter" @click.stop>
          <option value="">All</option>
          <option value="error">Error</option>
          <option value="warn">Warn</option>
          <option value="info">Info</option>
          <option value="debug">Debug</option>
        </select>
        <button v-if="!collapsed" class="icon-btn" title="Auto-scroll" @click.stop="autoScroll = !autoScroll" :class="{ active: autoScroll }">⬇</button>
        <button v-if="!collapsed" class="icon-btn" title="Clear" @click.stop="clear">🗑</button>
        <button class="icon-btn collapse-btn" @click.stop="collapsed = !collapsed">{{ collapsed ? '▲' : '▼' }}</button>
      </div>
    </div>
    <!-- Log list -->
    <div v-if="!collapsed" ref="listEl" class="log-list" @scroll="onScroll">
      <div
        v-for="(entry, i) in filteredLogs"
        :key="i"
        :class="['log-entry', 'level-' + entry.level]"
      >
        {{ formatLine(entry) }}
      </div>
      <div v-if="filteredLogs.length === 0" class="log-empty">No log entries yet.</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import type { LogEntry } from '../types/domain'
import { useLogs } from '../composables/useLogs'

const { logs, connected, clear } = useLogs()

const levelFilter = ref('')
const collapsed = ref(false)
const autoScroll = ref(true)
const panelHeight = ref(200)
const listEl = ref<HTMLElement | null>(null)

const filteredLogs = computed(() => {
  if (!levelFilter.value) return logs.value
  return logs.value.filter(l => l.level === levelFilter.value)
})

function formatLine(entry: LogEntry): string {
  try {
    const d = new Date(entry.timestamp)
    const hh = String(d.getHours()).padStart(2, '0')
    const mm = String(d.getMinutes()).padStart(2, '0')
    const ss = String(d.getSeconds()).padStart(2, '0')
    return `[${hh}:${mm}:${ss}] [${entry.level.toUpperCase()}] [${entry.source}] ${entry.message}`
  } catch {
    return `[${entry.level.toUpperCase()}] [${entry.source}] ${entry.message}`
  }
}

function onScroll() {
  if (!listEl.value) return
  const { scrollTop, clientHeight, scrollHeight } = listEl.value
  if (scrollTop + clientHeight < scrollHeight - 5) {
    autoScroll.value = false
  }
}

watch(
  () => filteredLogs.value.length,
  () => {
    if (autoScroll.value && listEl.value) {
      nextTick(() => {
        listEl.value!.scrollTop = listEl.value!.scrollHeight
      })
    }
  },
)

// Drag resize
let dragStartY = 0
let dragStartH = 0

function onDragStart(e: MouseEvent) {
  if (collapsed.value) return
  dragStartY = e.clientY
  dragStartH = panelHeight.value
  document.addEventListener('mousemove', onDragMove)
  document.addEventListener('mouseup', onDragEnd)
}

function onDragMove(e: MouseEvent) {
  const delta = dragStartY - e.clientY
  panelHeight.value = Math.max(80, Math.min(500, dragStartH + delta))
}

function onDragEnd() {
  document.removeEventListener('mousemove', onDragMove)
  document.removeEventListener('mouseup', onDragEnd)
}
</script>

<style scoped>
.log-panel {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  background: #1e1e2e;
  border-top: 2px solid #313244;
  z-index: 100;
  display: flex;
  flex-direction: column;
  font-size: 12px;
}
.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 10px;
  background: #181825;
  cursor: row-resize;
  user-select: none;
  flex-shrink: 0;
}
.log-header-left, .log-header-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.log-title { font-weight: 600; color: #cdd6f4; }
.status-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  display: inline-block;
}
.status-dot.connected { background: #a6e3a1; }
.status-dot.disconnected { background: #f38ba8; }
.log-count { color: #6c7086; }
.level-filter {
  background: #313244;
  border: 1px solid #45475a;
  color: #cdd6f4;
  border-radius: 4px;
  padding: 1px 4px;
  font-size: 11px;
}
.icon-btn {
  background: none;
  border: none;
  cursor: pointer;
  color: #6c7086;
  padding: 2px 4px;
  border-radius: 3px;
  font-size: 13px;
}
.icon-btn:hover, .icon-btn.active { color: #cdd6f4; background: #313244; }
.log-list {
  flex: 1;
  overflow-y: auto;
  padding: 4px 8px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
.log-entry {
  white-space: pre-wrap;
  word-break: break-all;
  padding: 1px 0;
  color: #cdd6f4;
}
.log-entry.level-warn { color: #f9e2af; }
.log-entry.level-error { color: #f38ba8; }
.log-entry.level-debug { color: #6c7086; }
.log-empty { color: #6c7086; font-style: italic; padding: 8px 0; }
</style>
