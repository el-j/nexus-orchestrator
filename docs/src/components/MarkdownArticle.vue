<!--
  A rendered markdown document, styled for the site. Props: html (trusted, build-time HTML from
  the repository's own markdown).
-->
<template>
  <article class="md-doc" data-testid="md-doc" v-html="html"></article>
</template>

<script setup lang="ts">
defineProps<{ html: string }>();
</script>

<style scoped>
.md-doc {
  color: #cbd5e1;
  line-height: 1.7;
}
.md-doc :deep(h1) {
  display: none; /* the page has its own title */
}
.md-doc :deep(h2) {
  font-size: 1.6rem;
  font-weight: 900;
  color: #f1f5f9;
  margin: 3rem 0 1.25rem;
  padding-top: 1rem;
  border-top: 1px solid rgb(255 255 255 / 0.08);
  scroll-margin-top: 6rem;
}
.md-doc :deep(h3) {
  font-size: 1.2rem;
  font-weight: 800;
  color: #e2e8f0;
  margin: 2.25rem 0 0.75rem;
  scroll-margin-top: 6rem;
}
.md-doc :deep(p),
.md-doc :deep(ul),
.md-doc :deep(ol) {
  margin: 0.75rem 0;
}
.md-doc :deep(ul) {
  list-style: disc;
  padding-left: 1.4rem;
}
.md-doc :deep(ol) {
  list-style: decimal;
  padding-left: 1.4rem;
}
.md-doc :deep(a) {
  color: #a78bfa;
}
.md-doc :deep(a:hover) {
  text-decoration: underline;
}
.md-doc :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.85em;
  background: #14141f;
  border: 1px solid rgb(255 255 255 / 0.06);
  border-radius: 0.35rem;
  padding: 0.1rem 0.35rem;
  color: #c4b5fd;
}
.md-doc :deep(pre) {
  background: #0d0d14;
  border: 1px solid rgb(255 255 255 / 0.08);
  border-radius: 0.75rem;
  padding: 1rem 1.25rem;
  overflow-x: auto;
  margin: 1rem 0;
}
.md-doc :deep(pre code) {
  background: none;
  border: 0;
  padding: 0;
  color: #e2e8f0;
  font-size: 0.85rem;
}
.md-doc :deep(table) {
  display: block;
  overflow-x: auto;
  border-collapse: collapse;
  margin: 1rem 0;
  font-size: 0.9rem;
}
.md-doc :deep(th),
.md-doc :deep(td) {
  border: 1px solid rgb(255 255 255 / 0.08);
  padding: 0.5rem 0.75rem;
  text-align: left;
  vertical-align: top;
}
.md-doc :deep(th) {
  background: #14141f;
  color: #e2e8f0;
}
.md-doc :deep(hr) {
  border: 0;
  border-top: 1px solid rgb(255 255 255 / 0.08);
  margin: 2rem 0;
}
.md-doc :deep(blockquote) {
  border-left: 3px solid #7c3aed;
  padding-left: 1rem;
  color: #94a3b8;
  margin: 1rem 0;
}
</style>
