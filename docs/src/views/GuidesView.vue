<!--
  Guides index: the how-to documents of the repository's docs/ folder, as cards.
-->
<template>
  <div class="max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <div class="mb-12">
      <div
        class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"
      >
        <UiIcon name="book" class="text-xs" />
        Guides
      </div>
      <h1 class="text-4xl font-black mb-4"><span class="gradient-text">Guides</span></h1>
      <p class="text-lg text-slate-400">
        How Nexus works, how to connect your agents and subscriptions, and how to shape the workflow
        of a project.
      </p>
    </div>
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-5">
      <RouterLink
        v-for="g in guides"
        :key="g.slug"
        :to="`/guides/${g.slug}`"
        class="rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/30 p-6 transition-all group"
      >
        <h2 class="font-bold text-white mb-2 group-hover:text-violet-300 transition-colors">
          {{ g.title }}
        </h2>
        <p class="text-sm text-slate-500">{{ g.summary }}</p>
      </RouterLink>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink } from 'vue-router';
import UiIcon from '../ui/UiIcon.vue';
import { guides } from '../data/guides';
</script>
