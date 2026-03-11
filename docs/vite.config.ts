import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'
import type {} from 'vite-ssg' // augments vite.UserConfig with ssgOptions

export default defineConfig({
  base: '/nexus-orchestrator/',
  plugins: [
    vue(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  ssgOptions: {
    // Produce route-name/index.html instead of route-name.html so that
    // GitHub Pages resolves /downloads/ → downloads/index.html without any
    // server-side rewrite rules.
    dirStyle: 'nested',
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
