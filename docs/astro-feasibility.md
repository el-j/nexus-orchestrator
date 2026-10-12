# Astro feasibility study (frontend and docs)

Issue #65. Question: before replacing the component library, would moving `frontend/` or `docs/` to
Astro be a better long-term architecture than staying on Vue 3 + Vite?

**Recommendation: no-go for the frontend, no-go (revisit at a checkpoint) for the docs.** The
reasoning and the evidence follow. The numbers marked _measured_ come from this repository; the
rest is analysis, not a prototype - no Astro build was run.

## Current baseline

|                         | `frontend/`                                                                                          | `docs/`                                                                              |
| ----------------------- | ---------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Stack                   | Vue 3 SPA, vue-router (history mode), Vite, Vitest, Playwright                                       | Vue 3 + vue-router, **vite-ssg** (static pre-render), Vitest                         |
| Delivery                | `go:embed all:build/frontend` in the Go binary, served by the daemon and loaded in the Wails webview | static HTML to GitHub Pages (6 routes)                                               |
| Data                    | runtime REST + SSE to `127.0.0.1:63987`, or Wails bindings                                           | none at runtime; `api-reference.md` and the MCP tool list are imported at build time |
| Size (_measured_, gzip) | JS 94.9 KiB, CSS 11.3 KiB                                                                            | JS 97.4 KiB, CSS 8.4 KiB                                                             |
| Tests                   | 782 unit tests at 100 % coverage + Playwright (mocked and real daemon)                               | 104 unit tests at 100 % coverage                                                     |

## Frontend: Astro vs Vue SPA

- **It is an application, not a content site.** Every view is client state: live task queue, SSE
  event stream, forms, drawers. Astro's value is shipping zero JS for static content; here
  essentially every page would be a single `client:load` Vue island, i.e. the same SPA behind an
  extra layer.
- **SSR/SSG gains nothing.** The page is embedded in a Go binary and talks to a loopback API; there
  is no SEO, no first-paint-from-server story, and no server to render on. Astro's SSR adapters
  would add a Node runtime the product deliberately does not have (single static binary).
- **Routing.** vue-router handles client navigation, route guards and the Wails webview's lack of
  a server for deep links (the daemon serves `index.html` as a fallback). Astro's file routing
  produces separate HTML pages; navigating between them reloads the shell and drops SSE/global
  state, unless View Transitions plus shared islands are used, which is more machinery than the
  current router.
- **MCP / API integration.** All calls go through `src/types/wails.ts` (Wails bindings or `fetch`).
  That layer is framework-independent and would carry over unchanged. Nothing about MCP or REST
  needs an Astro-specific adapter; the `go:embed` path (`build/frontend`) would have to be
  reconfigured (`outDir`).
- **DX / tests.** Vitest, Vue Test Utils and Playwright would mostly carry over; Astro components
  (`.astro`) are not testable with Vue Test Utils, so any logic moved into them loses the 100 %
  coverage gate's tooling. Stryker/vitest config would need re-validation.
- **Cost (estimate).** 6-10 engineer-days to move 12 views and 20 components behind islands and
  re-verify embedding, SSE and the Wails build, plus ongoing dual-framework knowledge. Risk:
  medium-high (embedding and webview behaviour). Benefit: none identified.

## Docs: Astro vs vite-ssg

- **Content fits Astro well, but it is already pre-rendered.** vite-ssg already emits static HTML
  for all 6 routes (`build/docs`, 20-57 KiB each) with the Vue components hydrating on top.
- **What Astro would add:** content collections for Markdown (today `api-reference.md` is imported
  with `?raw` and rendered with `marked`), zero-JS pages (the docs ship ~97 KiB gz of JS mostly for
  tabs, the mobile drawer and the scroll-reveal observer), built-in sitemap/RSS integrations.
- **What it would cost:** rewriting 6 views, 4 shared components and the layout as `.astro` or Vue islands, keeping
  the `docs/*.md` single-source tests (`internal/docscheck`) pointed at the new pipeline, and
  re-establishing the 100 % coverage tooling for `.astro` files. Estimate 3-5 engineer-days,
  low-medium risk (GitHub Pages deploy and the beta/stable base-path logic in `pages.yml`).
- **Verdict:** a real but small gain (less JS, simpler Markdown handling) that does not justify the
  work today.

## Go / no-go criteria and checkpoints

Revisit **docs** if any of these becomes true: the docs grow past ~25 pages or gain a blog/changelog
feed; Markdown becomes the primary authoring format for most pages; docs JS or build time becomes a
measured problem (budget in `scripts/check-bundle-budget.mjs` is breached).

Revisit the **frontend** only if the product gains a server-rendered, publicly indexable surface
(e.g. a hosted dashboard) - not for the embedded desktop/daemon UI.

Decision checkpoints: when the docs budget is breached in CI; at each minor release review of this
document; before any change that adds a Node runtime to the shipped artifacts.
