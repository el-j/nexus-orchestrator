# UI foundation

The frontend (`frontend/`) and the docs site (`docs/`) no longer depend on a third-party component
library. Both are built from a small set of internal primitives that live in `src/ui/` of each
workspace. This page describes the primitives, the standards they follow, and how the migration was
verified.

> Tracking: epic #64, foundation #66, services #67, shared layout #68, feature views #69, purges
> #70/#72, docs #71, regression #73, performance #74, cleanup #75, feasibility study #65.

## Primitives

| Primitive                  | Frontend | Docs | Replaces                                                  |
| -------------------------- | -------- | ---- | --------------------------------------------------------- |
| `UiIcon` + `icons.ts`      | yes      | yes  | the icon font (`<i class="pi pi-…">`)                     |
| `UiButton`                 | yes      | -    | `Button`                                                  |
| `UiSkeleton`               | yes      | -    | `Skeleton`                                                |
| `UiSelect`                 | yes      | -    | `Select`                                                  |
| `UiDrawer`                 | yes      | yes  | `Drawer`                                                  |
| `UiToaster` + `useToast()` | yes      | -    | `Toast`, `ToastService`, `useToast`                       |
| `UiTabs`                   | -        | yes  | `Tabs`, `TabList`, `Tab`, `TabPanels`, `TabPanel`         |
| `.ui-field` (CSS class)    | yes      | -    | `InputText`, `Textarea` (native `<input>` / `<textarea>`) |

Removed without a replacement because nothing used them: the confirmation service and dialog
(`useConfirm`, `ConfirmDialog`; `AppConfirmDialog.vue` already existed), and the `v-tooltip`
directive (the one use became a `title` + `aria-label`).

`UiToaster` is mounted once in `App.vue`. Call sites keep the same shape as before:

```ts
const toast = useToast();
toast.add({ severity: 'error', summary: 'Promote Failed', detail: String(e), life: 5000 });
```

In tests, mock `../composables/useToast` (as the component specs do) or read `activeToasts`.

## Design tokens and theming

`frontend/src/assets/tokens.css` defines colour, spacing, radius, type size, elevation and motion
tokens as CSS custom properties (`--ui-*`). The product ships the dark theme (`:root`); a light
theme is selected with `<html data-theme="light">`. The `.ui-field` form control and the focus ring
read the tokens, so a theme change needs no component edits.

## Keyboard, focus and ARIA standards

- Every interactive element is a native `<button>`, `<a>`, `<input>`, `<select>` or `<textarea>`,
  so it is reachable with Tab and activates with Enter / Space for free.
- Focus is always visible: a global `:focus-visible` outline (2px, offset 2px).
- Icon-only buttons carry an `aria-label`; icons themselves are `aria-hidden`.
- Form controls have an accessible name (`aria-label` on `UiSelect`, `placeholder` plus
  `aria-label` on inputs) and `aria-invalid` when validation fails.
- `UiDrawer` is a modal `role="dialog"` with `aria-modal`, an accessible name, Escape-to-close, a
  Tab / Shift+Tab focus trap, focus moved into the panel on open and restored to the opener on close.
- `UiTabs` follows the WAI-ARIA tabs pattern: `tablist` / `tab` / `tabpanel`, `aria-selected`,
  `aria-controls`, roving `tabindex`, Left / Right / Home / End. Panels stay in the DOM (hidden when
  inactive) so pre-rendered pages keep their content.
- Toasts are a polite live region; errors use `role="alert"`, the rest `role="status"`. Each toast
  can be dismissed with a button and removes itself after `life` ms (default 3 s).

## Motion policy

Transitions are short (0.2-0.25 s) and decorative. With `prefers-reduced-motion: reduce` every
animation and transition is reduced to ~0 ms (`tokens.css` in the frontend, `main.css` in the
docs), and the drawer's own transitions are disabled.

## Route parity (frontend)

Every route renders through the migrated primitives; route groups and what they use:

| Route group         | Routes                                                  | Uses                                                                                                    | Status   |
| ------------------- | ------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | -------- |
| Workspace dashboard | `/` (Mission Control), `/history`, `/backlog`, `/plans` | `UiButton`, `UiSkeleton`, `UiIcon`, `UiDrawer` (task detail), toasts, `UiSelect` + inputs (submit form) | migrated |
| Agents & activity   | `/agents`, `/projects/live-activity`                    | `UiIcon`, `UiDrawer`-style agent drawer                                                                 | migrated |
| Settings flows      | `/providers`, `/settings`, `/brain`, `/projects`        | `UiIcon`, toasts                                                                                        | migrated |
| App shell           | sidebar, log panel, error fallback                      | `UiIcon`, `UiToaster`                                                                                   | migrated |

Acceptance per route group: it renders the same content as before, every control is reachable and
operable with the keyboard, focus is visible, and its error paths raise a toast. These are covered
by the unit specs (100 % statement / branch / function / line coverage is enforced) and the
Playwright matrix below.

## Regression matrix

| Risk category      | What could regress                                                                 | Covered by                                                                            |
| ------------------ | ---------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Keyboard           | Tab order, Escape, Enter, arrow keys in tabs                                       | `ui/ui.spec.ts` (both workspaces), `e2e/ui-foundation.spec.ts`                        |
| Focus              | trap in the drawer, focus restore, visible ring                                    | `UiDrawer` specs, `Task detail drawer` e2e                                            |
| ARIA               | roles, names, `aria-invalid`, live regions                                         | `UiSelect` / `UiToaster` / `UiTabs` specs, e2e role queries (`getByRole`)             |
| Interaction parity | toast lifetime and dismissal, submit / cancel flows, select values keep their type | `useToast` specs, `Toasts` and `Task submit form` e2e, `TaskSubmitForm.spec.ts`       |
| Visual             | icon glyphs, tab and drawer styling                                                | glyph contact sheet and screenshots reviewed during the migration; no pixel-diff gate |
| Motion             | reduced motion                                                                     | `Motion policy` e2e                                                                   |
| Real data          | every view against a live daemon                                                   | `e2e/real/flows.spec.ts` (unchanged, runs in CI)                                      |

## Performance and bundle budget

Measured with `scripts/check-bundle-budget.mjs` (gzip, all `.js` and `.css` in the build output):

| Build        | Before (with the library) | After            | Budget enforced in CI |
| ------------ | ------------------------- | ---------------- | --------------------- |
| Frontend JS  | 189.8 KiB                 | 94.9 KiB (-50 %) | 110 KiB               |
| Frontend CSS | 13.2 KiB                  | 11.3 KiB (-14 %) | 13 KiB                |
| Docs JS      | 167.0 KiB                 | 97.4 KiB (-42 %) | 112 KiB               |
| Docs CSS     | 11.1 KiB                  | 8.4 KiB (-24 %)  | 10 KiB                |

The frontend entry chunk went from 412 kB to 58 kB (minified). The budgets sit about 15 % above the
measured sizes; CI fails the `Frontend Smoke Coverage` / `Docs Site Tests` job when one is exceeded.
Only build size is measured; no runtime (Lighthouse / interaction timing) benchmark exists yet.

## Guard against regressions

`scripts/check-no-ui-kit.mjs frontend docs` runs in CI and fails if the manifest or any source file
references the removed packages.
