import { configDefaults, defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    // Cold module transforms on a busy machine or CI runner can pass 5 s; a slow test is not a failing one.
    testTimeout: 20000,
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
      // mcpProxyEntry.ts only wires mcpProxy.ts to the process streams (the logic is tested there).
      exclude: [
        'src/**/*.test.ts',
        'src/**/*.d.ts',
        'src/test/**',
        'src/nexusTypes.ts',
        'src/mcpProxyEntry.ts',
      ],
      // Ratchet: raise these when coverage rises, never lower them.
      thresholds: { statements: 100, lines: 100, functions: 100, branches: 100 },
    },
  },
});
