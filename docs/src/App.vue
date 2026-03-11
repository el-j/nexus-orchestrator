<template>
  <div class="min-h-screen bg-[#050508] text-[#f0f0f8]">
    <AppNav />
    <main>
      <RouterView />
    </main>
    <AppFooter />
    <Toast position="bottom-right" />
  </div>
</template>

<script setup lang="ts">
import { RouterView, useRoute } from 'vue-router'
import { onMounted, watch, nextTick } from 'vue'
import AppNav from './components/AppNav.vue'
import AppFooter from './components/AppFooter.vue'
import Toast from 'primevue/toast'

const route = useRoute()

// Scroll reveal observer — re-runs on every route navigation
onMounted(() => {
  const observer = new IntersectionObserver((entries) => {
    entries.forEach(e => {
      if (e.isIntersecting) {
        e.target.classList.add('visible')
        observer.unobserve(e.target)
      }
    })
  }, { threshold: 0.1 })

  const observe = () => {
    // Only observe elements that have not yet become visible.
    // Each element is unobserved once it fires (see callback above), so
    // repeated calls on route change do not accumulate stale entries.
    document.querySelectorAll<Element>('.reveal:not(.visible)').forEach(el => observer.observe(el))
  }

  observe()

  // Re-observe after each route change so newly rendered .reveal elements are picked up
  watch(route, async () => {
    await nextTick()
    observe()
  })
})
</script>
