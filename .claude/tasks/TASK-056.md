# TASK-056 — Dashboard: full provider management UI + task submission form

**Plan:** PLAN-006  
**Role:** api (dashboard HTML)  
**Status:** todo  
**Dependencies:** TASK-055

## Goal

Upgrade the embedded dashboard in `httpapi/dashboard.go` to give users full control over:
1. **Provider panel** — shows `activeModel` tag, expandable model pills, add-provider modal, per-provider remove button
2. **Task submission form** — ModelID dropdown (populated from all providers' models), ProviderHint dropdown (from provider names)
3. **Badge coverage** — new `NO_PROVIDER` and `TOO_LARGE` badges

## Design constraints

- Pure HTML/CSS/vanilla JS only — zero external dependencies
- Must work inside the Wails webview (no CORS needed for `window.__WAILS__` calls)
- All API calls via `fetch` to the embedded HTTP server (`/api/*`)
- Accessible: semantic HTML, labels, keyboard-operable modals

## UI Components to add

### A. Provider panel — enriched cards

Each provider card shows:
- Colour dot (green/red)
- Provider name + kind badge
- Active model tag (e.g. `[llama3.2]`) when online
- Expand toggle → reveals model pills as `<span class="pill">modelId</span>`
- × remove button (calls `DELETE /api/providers/{name}`, skips built-in LMStudio/Ollama)

Below the provider list: **"+ Add Provider"** button → opens `#add-provider-dialog` modal

### B. Add-Provider modal (`#add-provider-dialog`)

Fields:
- Name (text, required)
- Kind (select: lmstudio / ollama / openaicompat / anthropic)
- Base URL (text, shown when kind ≠ anthropic)
- API Key (password, shown when kind = openaicompat | anthropic)
- Default Model (text)

On submit: `POST /api/providers` with JSON body → on success reload providers, close modal.

### C. Task submission form (collapsible panel)

Below stats, a **"Submit Task"** expandable panel with:
- Project Path (text)
- Target File (text)
- Instruction (textarea, 4 rows)
- Context Files (comma-separated text, hint: "file1.go, file2.go")
- Model ID (select, label: "Model (optional)") — `<option value="">Any active model</option>` + options from `GET /api/providers/{name}/models` for each online provider
- Provider Hint (select, label: "Prefer provider") — `<option value="">No preference</option>` + online provider names
- Submit button

On submit: `POST /api/tasks` with full JSON body → shows success toast with task ID.

### D. Badge additions

```js
const map = {
  QUEUED:'badge-queued', PROCESSING:'badge-processing', COMPLETED:'badge-completed',
  FAILED:'badge-failed', CANCELLED:'badge-cancelled',
  TOO_LARGE:'badge-toolarge', NO_PROVIDER:'badge-noprovider'
}
```

CSS for new badges:
```css
.badge-toolarge   { background: #451a03; color: #fdba74 }
.badge-noprovider { background: #2d1561; color: #c4b5fd }
```

### E. Task table — add ModelID column

New column after "Target File": "Model" showing `task.modelId || '—'`.

## Acceptance

- `go vet ./internal/adapters/inbound/httpapi/...` passes (Go side unchanged)
- No JS console errors on page load
- Provider registration round-trip works via `POST /api/providers`
- Task table shows new columns and badges
