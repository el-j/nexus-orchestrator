<template>
  <div class="relative group">
    <pre class="!mt-0"><code :class="`language-${language}`">{{ code }}</code></pre>
    <button
      @click="copy"
      class="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity
             px-2 py-1 text-xs rounded bg-white/10 hover:bg-violet-600/40 text-slate-300 font-mono cursor-pointer"
    >
      {{ copied ? '✓ copied' : 'copy' }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{ code: string; language?: string }>()
const copied = ref(false)

async function copy() {
  try {
    await navigator.clipboard.writeText(props.code)
  } catch {
    const el = document.createElement('textarea')
    el.value = props.code
    document.body.appendChild(el)
    el.select()
    document.execCommand('copy')
    document.body.removeChild(el)
  }
  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}
</script>
