import { defineConfig } from 'vitest/config';
import vue from '@vitejs/plugin-vue';
import { resolve } from 'path';

export default defineConfig({
  plugins: [vue()],
  resolve: { alias: { '@': resolve(__dirname, './src') } },
  test: {
    // Cold module transforms on a busy machine or CI runner can pass 5 s; a slow test is not a failing one.
    testTimeout: 20000,
    environment: 'jsdom',
    globals: true,
    include: ['src/**/*.spec.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      include: ['src/**/*.{ts,vue}'],
      exclude: ['src/**/*.spec.ts', 'src/**/*.d.ts', 'src/test/**'],
      thresholds: { statements: 100, lines: 100, functions: 100, branches: 100 },
    },
  },
});
