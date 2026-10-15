<!--
  One guide of the site: a markdown file of the repository's docs/ folder with its table of
  contents. The guide is chosen by the route (meta.slug); links between guides are site pages.
-->
<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-12">
      <aside class="lg:col-span-1">
        <div class="sticky top-24 space-y-1 max-h-[calc(100vh-8rem)] overflow-y-auto pr-2">
          <RouterLink to="/guides" class="block text-xs text-violet-400 hover:underline mb-4"
            >← All guides</RouterLink
          >
          <h4 class="text-xs font-semibold uppercase tracking-wider text-slate-500 mb-4">
            On this page
          </h4>
          <nav class="space-y-1" aria-label="On this page">
            <a
              v-for="item in toc"
              :key="item.id"
              :href="`#${item.id}`"
              :class="[
                'block text-sm transition-colors py-0.5 cursor-pointer hover:text-violet-400',
                item.depth === 2 ? 'text-slate-300 font-semibold mt-3' : 'text-slate-500 pl-3',
              ]"
              @click.prevent="scrollToId(item.id)"
              >{{ item.label }}</a
            >
          </nav>
        </div>
      </aside>

      <main class="lg:col-span-3">
        <div class="mb-10">
          <div
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"
          >
            <UiIcon name="book" class="text-xs" />
            Guide
          </div>
          <h1 class="text-4xl font-black mb-4">
            <span class="gradient-text">{{ guide.title }}</span>
          </h1>
          <p class="text-lg text-slate-400">{{ guide.summary }}</p>
          <p class="mt-4 text-sm text-slate-500">
            Rendered from
            <a
              class="text-violet-400 hover:underline"
              :href="`https://github.com/el-j/nexus-orchestrator/blob/main/docs/${guide.slug}.md`"
              target="_blank"
              rel="noopener"
              >docs/{{ guide.slug }}.md</a
            >
            in the repository, so it cannot drift from it.
          </p>
        </div>
        <MarkdownArticle :html="html" />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { RouterLink, useRoute } from 'vue-router';
import UiIcon from '../ui/UiIcon.vue';
import MarkdownArticle from '../components/MarkdownArticle.vue';
import { guideLink, guides } from '../data/guides';
import { renderMarkdown } from '../utils/markdown';
import { scrollToId } from '../utils/scroll';

const route = useRoute();
// The router only sends known slugs here (one route per guide).
const guide = guides.find((g) => g.slug === route.meta.slug)!;
const { html, toc } = renderMarkdown(guide.source, {
  linkHref: (href) => guideLink(href, import.meta.env.BASE_URL),
});
</script>
