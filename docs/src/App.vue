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
import { RouterView } from 'vue-router'
import { onMounted } from 'vue'
import AppNav from './components/AppNav.vue'
import AppFooter from './components/AppFooter.vue'
import Toast from 'primevue/toast'

// Scroll reveal observer
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
    document.querySelectorAll('.reveal').forEach(el => observer.observe(el))
  }

  observe()
  // Re-observe on route change
  const interval = setInterval(observe, 500)
  setTimeout(() => clearInterval(interval), 3000)
})
</script>
