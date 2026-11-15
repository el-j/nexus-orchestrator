# Mutation testing

Mutation testing checks that the tests would notice a bug, not just that they execute the code. It
changes the source in small ways (flips `<` to `<=`, negates a condition, swaps `+` for `-`) and
requires at least one test to fail for each change. A mutant that survives points at a missing or
weak assertion.

**It runs on your machine only.** No GitHub workflow runs it: a full run takes minutes per package
and would consume workflow minutes. Do not add it to `.github/workflows`.

## Running it

```sh
make mutation                                  # every Go package with logic + every TypeScript package
scripts/mutation.sh go                         # Go only
scripts/mutation.sh go internal/adapters/outbound/fs_writer   # chosen Go packages
scripts/mutation.sh ts                         # frontend, docs, vscode-extension, github-action
```

| Language   | Tool                                                | Configuration                                                                                                                     | Gate                                                           |
| ---------- | --------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------- |
| Go         | [gremlins](https://github.com/go-gremlins/gremlins) | `scripts/mutation.sh` (`--integration`, `--timeout-coefficient ${MUTATION_TIMEOUT_COEFF:-3}`; defaults: every package with logic) | `MUTATION_MIN_EFFICACY`, default 70 % killed of killed + lived |
| TypeScript | [StrykerJS](https://stryker-mutator.io)             | `stryker.config.json` in each package                                                                                             | its `thresholds.break`                                         |

Install gremlins once with `go install github.com/go-gremlins/gremlins/cmd/gremlins@latest`.

`scripts/mutation.sh` passes `--integration` so gremlins runs tests across the module rather than
only the mutated package's own `*_test.go` (for example, `internal/core/domain` types are also
exercised by `internal/core/services` and adapter tests), and `--timeout-coefficient` (default `3`,
override with `MUTATION_TIMEOUT_COEFF`) so sub-second cancellation/watchdog tests do not produce
spurious timeout mutants.

## Reading the result and accepted equivalent mutants

- **Killed**: a test failed, which is the desired outcome.
- **Lived**: no test failed. Add or tighten an assertion, or document why the mutant is equivalent
  (the changed code behaves identically). Common equivalent mutants in this codebase:
  - Boundary mutations (`<` vs `<=` on `0` or `len(slice)`) where the zero/empty case takes an
    identical no-op path (for example `maxTokens <= 0` defaulting to the same budget, or
    `cut > 0` before appending an empty slice).
  - Arithmetic/backoff increments in retry or polling loops where the test clock or mock channel
    advances past both bounds.
- **Not covered**: no test runs that line, so fix coverage first.
- **Timed out**: the mutant made the code loop or hang. Gremlins counts it as detected-by-timeout
  but reports it separately; it is noisy for loops over a changed bound.

Run it on the package you are changing before opening a pull request that touches core logic, and
treat a lived mutant in new code as a review comment on your own change.

## Measured results

Measured with `--integration --timeout-coefficient 10` on 2026-11-07. A full run of all packages takes
hours, so the table holds what has been measured; run the rest with the commands above and add the
number here.

| Package                | Killed | Lived | Efficacy | Notes                                                                                                                                                                                                                      |
| ---------------------- | -----: | ----: | -------: | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/core/domain` |     81 |     3 |   96.4 % | Was 85.4 % (13 lived); `boundaries_test.go` tests every limit on both sides. The 3 left are equivalent: the `i+1` of a rule number in an error message, and a `key != ""` guard whose empty case finds nothing either way. |

gremlins also reports 26 mutants as "not covered" in `domain`; its coverage run looks at the package's own
tests, while the lines are exercised by the integration run, so treat that count as noise.

**Docs site (StrykerJS), measured 2026-11-07:** over all of `src` the score was 43.3 %, but 90 % of the
mutants sat in four files that are only data (`architecture.ts`, `downloads.ts`, `mcpTools.ts`,
`mcpIntegration.ts`: page text and tables whose changes a unit test cannot meaningfully "kill"; the
docscheck drift tests compare them with the daemon). They are excluded; the logic that remains (router,
markdown rendering, guides, scroll) scores 94.3 %, and the break threshold is 90.
