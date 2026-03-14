import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'

export default defineConfig({
  base: './',
  plugins: [
    vue(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  // Dev server: proxy /api and /mcp to the running nexus-daemon on :63987
  server: {
    port: 63989,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:63987',
        changeOrigin: true,
      },
      '/mcp': {
        target: 'http://127.0.0.1:63987',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../build/frontend',
    emptyOutDir: true,
  },
})
