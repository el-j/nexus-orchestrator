import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import tailwindcss from '@tailwindcss/vite';
import { resolve } from 'path';

export default defineConfig({
  base: './',
  plugins: [vue(), tailwindcss()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  // Dev server: proxy /api and /.well-known to HTTP API on :63987, /mcp to MCP server on :63988
  server: {
    port: 63989,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:63987',
        changeOrigin: true,
      },
      '/mcp': {
        target: 'http://127.0.0.1:63988',
        changeOrigin: true,
      },
      '/.well-known': {
        target: 'http://127.0.0.1:63987',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: '../build/frontend',
    emptyOutDir: true,
  },
});
