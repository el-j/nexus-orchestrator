<!--
  API Reference page: the REST and MCP reference rendered from docs/api-reference.md.
-->
<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-12">
      <!-- Sidebar TOC -->
      <aside class="lg:col-span-1">
        <div class="sticky top-24 space-y-1 max-h-[calc(100vh-8rem)] overflow-y-auto pr-2">
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

      <!-- Main content -->
      <main class="lg:col-span-3">
        <div class="mb-10">
          <div
            class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-4"
          >
            <UiIcon name="server" class="text-xs" />
            HTTP REST + MCP
          </div>
          <h1 class="text-4xl font-black mb-4"><span class="gradient-text">API Reference</span></h1>
          <p class="text-lg text-slate-400">
            Complete reference for all HTTP REST API endpoints and MCP tools.
          </p>
          <div class="mt-4 flex flex-wrap gap-3">
            <span
              class="px-3 py-1 rounded-lg bg-[#0d0d14] border border-white/8 text-xs text-slate-400 font-mono"
              >Base URL: http://localhost:63987</span
            >
            <span
              class="px-3 py-1 rounded-lg bg-[#0d0d14] border border-white/8 text-xs text-slate-400 font-mono"
              >MCP: http://localhost:63988/mcp</span
            >
          </div>
          <p class="mt-4 text-sm text-slate-500">
            Generated from
            <a
              class="text-violet-400 hover:underline"
              href="https://github.com/el-j/nexus-orchestrator/blob/main/docs/api-reference.md"
              target="_blank"
              rel="noopener"
              >docs/api-reference.md</a
            >
            in the repository, so it cannot drift from it.
          </p>
        </div>

        <article class="api-doc" data-testid="api-doc" v-html="html"></article>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import UiIcon from '../ui/UiIcon.vue';
import { renderMarkdown } from '../utils/markdown';
import { scrollToId } from '../utils/scroll';
import apiReference from '../../api-reference.md?raw';

// Rendered once, at build time for the static site and on first load in dev.
const { html, toc } = renderMarkdown(apiReference);
</script>

<style scoped>
.api-doc {
  color: #cbd5e1;
  line-height: 1.7;
}
.api-doc :deep(h1) {
  display: none; /* the page already has its own title */
}
.api-doc :deep(h2) {
  font-size: 1.6rem;
  font-weight: 900;
  color: #f1f5f9;
  margin: 3rem 0 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid rgb(255 255 255 / 0.08);
}
.api-doc :deep(h3) {
  font-size: 1.2rem;
  font-weight: 800;
  color: #e2e8f0;
  margin: 2.25rem 0 0.75rem;
  scroll-margin-top: 6rem;
}
.api-doc :deep(h4) {
  font-size: 1rem;
  font-weight: 700;
  color: #cbd5e1;
  margin: 1.5rem 0 0.5rem;
}
.api-doc :deep(p),
.api-doc :deep(ul),
.api-doc :deep(ol) {
  margin: 0.75rem 0;
}
.api-doc :deep(ul) {
  list-style: disc;
  padding-left: 1.4rem;
}
.api-doc :deep(ol) {
  list-style: decimal;
  padding-left: 1.4rem;
}
.api-doc :deep(a) {
  color: #a78bfa;
}
.api-doc :deep(a:hover) {
  text-decoration: underline;
}
.api-doc :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85em;
  background: #14141f;
  border: 1px solid rgb(255 255 255 / 0.06);
  border-radius: 0.35rem;
  padding: 0.1rem 0.35rem;
  color: #c4b5fd;
}
.api-doc :deep(pre) {
  background: #0d0d14;
  border: 1px solid rgb(255 255 255 / 0.08);
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  overflow-x: auto;
  margin: 1rem 0;
}
.api-doc :deep(pre code) {
  background: none;
  border: 0;
  padding: 0;
  color: #e2e8f0;
  font-size: 0.85rem;
}
.api-doc :deep(table) {
  display: block;
  overflow-x: auto;
  border-collapse: collapse;
  margin: 1rem 0;
  font-size: 0.9rem;
}
.api-doc :deep(th),
.api-doc :deep(td) {
  border: 1px solid rgb(255 255 255 / 0.08);
  padding: 0.5rem 0.75rem;
  text-align: left;
  vertical-align: top;
}
.api-doc :deep(th) {
  background: #14141f;
  color: #e2e8f0;
}
.api-doc :deep(hr) {
  border: 0;
  border-top: 1px solid rgb(255 255 255 / 0.08);
  margin: 2rem 0;
}
.api-doc :deep(blockquote) {
  border-left: 3px solid #7c3aed;
  padding-left: 1rem;
  color: #94a3b8;
  margin: 1rem 0;
}
</style>
