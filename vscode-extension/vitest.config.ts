import { configDefaults, defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'node',
    globals: true,
    // host-test runs inside a real VS Code host (npm run test:host), not under vitest.
    exclude: [...configDefaults.exclude, 'host-test/**', '.vscode-test/**'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov'],
      // Measure ALL extension source, not a hand-picked subset.
      include: ['src/**/*.ts'],
      // nexusTypes.ts holds only type declarations, so it has no runtime code to cover.
      exclude: ['src/**/*.test.ts', 'src/**/*.d.ts', 'src/test/**', 'src/nexusTypes.ts'],
      // Ratchet: raise these when coverage rises, never lower them.
      thresholds: { statements: 100, lines: 100, functions: 100, branches: 100 },
    },
  },
});
