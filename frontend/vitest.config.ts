import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    exclude: ['e2e/**', 'playwright-report/**', 'node_modules/**', 'dist/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Measure ALL application source, not a hand-picked pair of files.
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/**/*.{spec,test}.ts',
        'src/test/**',
        'src/wailsjs/**', // generated Wails bindings
        'src/**/*.d.ts',
      ],
    },
  },
});
