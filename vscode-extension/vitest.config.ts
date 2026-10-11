import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    globals: true,
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Measure ALL extension source, not a hand-picked subset.
      include: ['src/**/*.ts'],
      exclude: ['src/**/*.test.ts', 'src/**/*.d.ts', 'src/test/**'],
      // Ratchet: raise these when coverage rises, never lower them.
      thresholds: { statements: 99, lines: 99, functions: 98, branches: 97 },
    },
  },
});
