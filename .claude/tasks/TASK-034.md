---
id: TASK-034
title: Embedded web dashboard at GET /ui with live queue view
role: devops
planId: PLAN-003
status: todo
dependencies: [TASK-032]
createdAt: 2026-03-09T15:00:00.000Z
---

## Context

nexusOrchestrator has no web UI. This task embeds a self-contained HTML dashboard served at `GET /ui` (and `GET /` → redirect to `/ui`) by the existing HTTP API server. There is NO frontend build process — the entire UI is a Go `text/template` embedded in the binary. The dashboard auto-refreshes every 2 seconds using vanilla JavaScript `fetch`.

This gives immediate visual feedback when dogfooding PLAN-002 tasks through the running daemon.

## Files to Modify / Create

- `internal/adapters/inbound/httpapi/server.go` — add `/ui` route + redirect
- `internal/adapters/inbound/httpapi/dashboard.go` — HTML template + handler

## Design Specification

### Layout
- Dark theme (background: #09090b, surface: #18181b, text: #f4f4f5)
- Indigo accent: #6366f1
- Two-column layout on wide screens: stats row (4 cards) → providers (left) + queue table (right)
- Fully responsive fallback to single column on narrow viewports

### Stat Cards (top row)
- "Queue Depth" — count of QUEUED tasks
- "Processing" — count of PROCESSING tasks
- "Providers Online" — count of active providers
- "Completed" — total count from history (GET /api/tasks — extended in TASK-015; for now, 0 until that endpoint exists)

### Provider Cards
One card per provider from `GET /api/providers`:
- Provider name + colour badge (green=active, red=offline)
- Pulsing dot for active state (CSS animation)

### Task Queue Table
Columns: Status badge, ID (first 8 chars), Project (last 2 segments), Target File, Submitted (relative time)
Data from `GET /api/tasks` (QUEUED+PROCESSING only)
Empty state: "Queue is empty — submit tasks via nexus-submit or POST /api/tasks"

### Auto-refresh
JavaScript `setInterval` polling `GET /api/tasks` and `GET /api/providers` every 2000ms.
On SSE availability (TASK-035), fall back to EventSource.

## Implementation: internal/adapters/inbound/httpapi/dashboard.go

```go
package httpapi

import (
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dash").Parse(dashboardTemplateHTML))

const dashboardTemplateHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>nexusOrchestrator</title>
  <style>
    *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0 }
    :root {
      --bg: #09090b; --surface: #18181b; --surface2: #27272a;
      --border: #3f3f46; --text: #f4f4f5; --muted: #a1a1aa;
      --indigo: #6366f1; --green: #22c55e; --red: #ef4444;
      --yellow: #eab308; --font: system-ui, sans-serif;
    }
    body { background: var(--bg); color: var(--text); font-family: var(--font); min-height: 100vh }
    header { background: var(--surface); border-bottom: 1px solid var(--border);
             padding: 0.75rem 1.5rem; display: flex; align-items: center; gap: 0.75rem }
    header h1 { font-size: 1rem; font-weight: 700; color: var(--indigo) }
    header .subtitle { font-size: 0.75rem; color: var(--muted) }
    .container { max-width: 1400px; margin: 0 auto; padding: 1.5rem }
    .stats { display: grid; grid-template-columns: repeat(auto-fit,minmax(160px,1fr)); gap: 1rem; margin-bottom: 1.5rem }
    .stat-card { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; padding: 1rem }
    .stat-card .label { font-size: 0.7rem; text-transform: uppercase; letter-spacing: .05em; color: var(--muted); margin-bottom: 0.5rem }
    .stat-card .value { font-size: 1.75rem; font-weight: 700 }
    .grid2 { display: grid; grid-template-columns: 280px 1fr; gap: 1.25rem }
    @media(max-width:768px){ .grid2 { grid-template-columns: 1fr } }
    .panel { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; overflow: hidden }
    .panel-header { padding: 0.75rem 1rem; border-bottom: 1px solid var(--border);
                    font-size: 0.8rem; font-weight: 600; text-transform: uppercase; letter-spacing: .05em; color: var(--muted) }
    .provider-card { padding: 0.75rem 1rem; border-bottom: 1px solid var(--border); display: flex; align-items: center; gap: 0.5rem }
    .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0 }
    .dot-green { background: var(--green); box-shadow: 0 0 6px var(--green); animation: pulse 2s infinite }
    .dot-red   { background: var(--red) }
    @keyframes pulse { 0%,100%{opacity:1}50%{opacity:.4} }
    table { width: 100%; border-collapse: collapse; font-size: 0.8rem }
    th { text-align: left; padding: 0.6rem 0.75rem; color: var(--muted); font-weight: 500;
         border-bottom: 1px solid var(--border); white-space: nowrap }
    td { padding: 0.55rem 0.75rem; border-bottom: 1px solid var(--border); vertical-align: middle }
    tr:last-child td { border-bottom: none }
    .badge { display: inline-block; padding: 0.15rem 0.5rem; border-radius: 63987px; font-size: 0.7rem; font-weight: 600 }
    .badge-queued     { background: #1e3a5f; color: #93c5fd }
    .badge-processing { background: #422006; color: #fde68a }
    .badge-completed  { background: #14532d; color: #86efac }
    .badge-failed     { background: #450a0a; color: #fca5a5 }
    .mono { font-family: monospace; font-size: 0.75rem; color: var(--muted) }
    .empty { padding: 2rem; text-align: center; color: var(--muted); font-size: 0.85rem }
    #refresh-indicator { font-size:0.7rem; color:var(--muted); margin-left:auto }
  </style>
</head>
<body>
<header>
  <h1>nexusOrchestrator</h1>
  <span class="subtitle">Live Queue Dashboard</span>
  <span id="refresh-indicator">–</span>
</header>
<div class="container">
  <div class="stats">
    <div class="stat-card"><div class="label">Queued</div><div class="value" id="stat-queued">–</div></div>
    <div class="stat-card"><div class="label">Processing</div><div class="value" id="stat-processing">–</div></div>
    <div class="stat-card"><div class="label">Providers Online</div><div class="value" id="stat-providers">–</div></div>
    <div class="stat-card"><div class="label">Total in Queue</div><div class="value" id="stat-total">–</div></div>
  </div>
  <div class="grid2">
    <div class="panel">
      <div class="panel-header">LLM Providers</div>
      <div id="providers-list"><div class="empty">Loading…</div></div>
    </div>
    <div class="panel">
      <div class="panel-header">Task Queue</div>
      <div id="queue-container">
        <table>
          <thead><tr><th>Status</th><th>ID</th><th>Project</th><th>Target File</th><th>Submitted</th></tr></thead>
          <tbody id="queue-body"><tr><td colspan="5" class="empty">Loading…</td></tr></tbody>
        </table>
      </div>
    </div>
  </div>
</div>
<script>
const rtf = new Intl.RelativeTimeFormat('en', {numeric:'auto'})
function reltime(iso) {
  const d = (new Date(iso) - Date.now()) / 1000
  if (Math.abs(d)<60) return rtf.format(Math.round(d),'seconds')
  if (Math.abs(d)<3600) return rtf.format(Math.round(d/60),'minutes')
  return rtf.format(Math.round(d/3600),'hours')
}
function badge(s) {
  const cls = {QUEUED:'badge-queued',PROCESSING:'badge-processing',COMPLETED:'badge-completed',FAILED:'badge-failed'}[s]||'badge-queued'
  return '<span class="badge '+cls+'">'+s+'</span>'
}
function projShort(p) { return p.split('/').slice(-2).join('/') }

async function refresh() {
  try {
    const [tasks, providers] = await Promise.all([
      fetch('/api/tasks').then(r=>r.json()),
      fetch('/api/providers').then(r=>r.json()),
    ])
    // Stats
    const q = tasks.filter(t=>t.status==='QUEUED').length
    const p = tasks.filter(t=>t.status==='PROCESSING').length
    const on = providers.filter(p=>p.active).length
    document.getElementById('stat-queued').textContent = q
    document.getElementById('stat-processing').textContent = p
    document.getElementById('stat-providers').textContent = on+'/'+providers.length
    document.getElementById('stat-total').textContent = tasks.length
    // Providers
    document.getElementById('providers-list').innerHTML = providers.length === 0
      ? '<div class="empty">No providers configured.</div>'
      : providers.map(p=>'<div class="provider-card"><span class="dot '+(p.active?'dot-green':'dot-red')+'"></span><span>'+p.name+'</span></div>').join('')
    // Queue
    const tbody = document.getElementById('queue-body')
    if (tasks.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty">Queue is empty — submit tasks via nexus-submit or POST /api/tasks</td></tr>'
    } else {
      tbody.innerHTML = tasks.map(t=>'<tr>'
        +'<td>'+badge(t.status)+'</td>'
        +'<td class="mono" title="'+t.id+'">'+t.id.slice(0,8)+'</td>'
        +'<td class="mono">'+projShort(t.projectPath)+'</td>'
        +'<td class="mono">'+(t.targetFile||'—')+'</td>'
        +'<td class="mono" title="'+t.createdAt+'">'+reltime(t.createdAt)+'</td>'
        +'</tr>').join('')
    }
    document.getElementById('refresh-indicator').textContent = 'updated '+new Date().toLocaleTimeString()
  } catch(e) {
    document.getElementById('refresh-indicator').textContent = 'error: '+e.message
  }
}
refresh()
setInterval(refresh, 2000)
</script>
</body>
</html>`

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := dashboardTmpl.Execute(w, nil); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
```

## Wiring in server.go

Add these routes to the chi router in `StartServer`:
```go
// Dashboard — GET /ui (and / → /ui redirect)
r.Get("/", func(w http.ResponseWriter, r *http.Request) {
    http.Redirect(w, r, "/ui", http.StatusFound)
})
r.Get("/ui", s.handleUI)
```

Note: `StartServer` is currently a plain function (not a method). Refactor to a `Server` struct with `handleUI` as a method, OR keep as a function and add the handler inline. The refactor to a struct is cleaner and enables TASK-035 (SSE) to share state.

Recommended refactor:
```go
type Server struct {
    orch ports.Orchestrator
    mux  *chi.Mux
}

func NewServer(orch ports.Orchestrator) *Server {
    s := &Server{orch: orch}
    // ... register routes ...
    return s
}

func StartServer(ctx context.Context, orch ports.Orchestrator, addr string) error {
    s := NewServer(orch)
    // ... create http.Server with s as handler ...
}
```

## Acceptance Criteria

- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=1 go build ./...` exits 0
- [ ] `CGO_ENABLED=1 go test -race -count=1 ./...` exits 0
- [ ] `GET http://localhost:63987/ui` returns HTTP 200 with `Content-Type: text/html`
- [ ] `GET http://localhost:63987/` redirects to `/ui`
- [ ] Dashboard shows provider cards and queue table
- [ ] JavaScript polls every 2 seconds; queue updates appear without page reload
- [ ] Status badges render with correct colours (indigo=queued, yellow=processing, green=completed, red=failed)
- [ ] Works in Chromium-based Wails WebView (no external CDN dependencies)
- [ ] No `text/template` injection issues — all dynamic data comes from JS fetch, not `template.Execute`

## Anti-patterns to Avoid

- NEVER load CSS/JS from CDN (no internet in air-gapped environments, Wails WebView restrictions)
- NEVER use `template.HTML` unsafely — user content must be sanitised before injection
- NEVER embed large base64 images — keep the template minimal
- NEVER block the HTTP handler with template parsing — parse once at package init
- NEVER use innerHTML with unsanitised task data — escape all user-provided strings
