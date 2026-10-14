# Mutation testing

Mutation testing checks that the tests would notice a bug, not just that they execute the code. It
changes the source in small ways (flips `<` to `<=`, negates a condition, swaps `+` for `-`) and
requires at least one test to fail for each change. A mutant that survives points at a missing or
weak assertion.

**It runs on your machine only.** No GitHub workflow runs it: a full run takes minutes per package
and would consume workflow minutes. Do not add it to `.github/workflows`.

## Running it

```sh
make mutation                                  # Go core + every TypeScript package
scripts/mutation.sh go                         # Go only
scripts/mutation.sh go internal/adapters/outbound/fs_writer   # chosen Go packages
scripts/mutation.sh ts                         # frontend, vscode-extension, github-action
```

| Language   | Tool                                                | Configuration                                                                             | Gate                                                           |
| ---------- | --------------------------------------------------- | ----------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| Go         | [gremlins](https://github.com/go-gremlins/gremlins) | `scripts/mutation.sh` (default packages `internal/core/domain`, `internal/core/services`) | `MUTATION_MIN_EFFICACY`, default 70 % killed of killed + lived |
| TypeScript | [StrykerJS](https://stryker-mutator.io)             | `stryker.config.json` in each package                                                     | its `thresholds.break`                                         |

Install gremlins once with `go install github.com/go-gremlins/gremlins/cmd/gremlins@latest`.

## Reading the result

- **Killed**: a test failed, which is the desired outcome.
- **Lived**: no test failed. Add or tighten an assertion, or document why the mutant is equivalent
  (the changed code behaves identically).
- **Not covered**: no test runs that line, so fix coverage first.
- **Timed out**: the mutant made the code loop or hang. Gremlins counts it as detected-by-timeout
  but reports it separately; it is noisy for loops over a changed bound.

Run it on the package you are changing before opening a pull request that touches core logic, and
treat a lived mutant in new code as a review comment on your own change.
