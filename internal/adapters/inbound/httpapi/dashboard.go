package httpapi

import (
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dashboard").Parse(dashboardTemplateHTML))

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if err := dashboardTmpl.Execute(w, nil); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

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
    .badge { display: inline-block; padding: 0.15rem 0.5rem; border-radius: 9999px; font-size: 0.7rem; font-weight: 600 }
    .badge-queued     { background: #1e3a5f; color: #93c5fd }
    .badge-processing { background: #422006; color: #fde68a }
    .badge-completed  { background: #14532d; color: #86efac }
    .badge-failed     { background: #450a0a; color: #fca5a5 }
    .badge-cancelled  { background: #27272a; color: #a1a1aa }
    .mono { font-family: monospace; font-size: 0.75rem; color: var(--muted) }
    .empty { padding: 2rem; text-align: center; color: var(--muted); font-size: 0.85rem }
    #refresh-ts { font-size:0.7rem; color:var(--muted); margin-left:auto }
  </style>
</head>
<body>
<header>
  <h1>nexusOrchestrator</h1>
  <span class="subtitle">Live Queue Dashboard</span>
  <span id="refresh-ts">–</span>
</header>
<div class="container">
  <div class="stats">
    <div class="stat-card"><div class="label">Queued</div><div class="value" id="stat-queued">–</div></div>
    <div class="stat-card"><div class="label">Processing</div><div class="value" id="stat-processing">–</div></div>
    <div class="stat-card"><div class="label">Providers Online</div><div class="value" id="stat-providers">–</div></div>
    <div class="stat-card"><div class="label">Total in View</div><div class="value" id="stat-total">–</div></div>
  </div>
  <div class="grid2">
    <div class="panel">
      <div class="panel-header">LLM Providers</div>
      <div id="providers-list"><div class="empty">Loading…</div></div>
    </div>
    <div class="panel">
      <div class="panel-header">Task Queue</div>
      <table>
        <thead><tr><th>Status</th><th>ID</th><th>Project</th><th>Target File</th><th>Submitted</th></tr></thead>
        <tbody id="queue-body"><tr><td colspan="5" class="empty">Loading…</td></tr></tbody>
      </table>
    </div>
  </div>
</div>
<script>
function esc(s) {
  return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;')
}
const rtf = new Intl.RelativeTimeFormat('en', {numeric:'auto'})
function reltime(iso) {
  const d = (new Date(iso) - Date.now()) / 1000
  if (Math.abs(d) < 60) return rtf.format(Math.round(d), 'seconds')
  if (Math.abs(d) < 3600) return rtf.format(Math.round(d/60), 'minutes')
  return rtf.format(Math.round(d/3600), 'hours')
}
function badge(s) {
  const map = {QUEUED:'badge-queued',PROCESSING:'badge-processing',COMPLETED:'badge-completed',FAILED:'badge-failed',CANCELLED:'badge-cancelled'}
  return '<span class="badge '+(map[s]||'badge-queued')+'">'+esc(s)+'</span>'
}
function projShort(p) { return esc(p.split('/').slice(-2).join('/')) }

async function refresh() {
  try {
    const [tasks, providers] = await Promise.all([
      fetch('/api/tasks').then(r=>r.json()),
      fetch('/api/providers').then(r=>r.json()),
    ])
    const q = tasks.filter(t=>t.status==='QUEUED').length
    const p = tasks.filter(t=>t.status==='PROCESSING').length
    const on = providers.filter(p=>p.active).length
    document.getElementById('stat-queued').textContent = q
    document.getElementById('stat-processing').textContent = p
    document.getElementById('stat-providers').textContent = on+'/'+providers.length
    document.getElementById('stat-total').textContent = tasks.length
    document.getElementById('providers-list').innerHTML = providers.length === 0
      ? '<div class="empty">No providers configured.</div>'
      : providers.map(p=>'<div class="provider-card"><span class="dot '+(p.active?'dot-green':'dot-red')+'"></span><span>'+esc(p.name)+'</span></div>').join('')
    const tbody = document.getElementById('queue-body')
    if (tasks.length === 0) {
      tbody.innerHTML = '<tr><td colspan="5" class="empty">Queue is empty — submit tasks via nexus-submit or POST /api/tasks</td></tr>'
    } else {
      tbody.innerHTML = tasks.map(t=>'<tr><td>'+badge(t.status)+'</td><td class="mono" title="'+esc(t.id)+'">'+esc(t.id.slice(0,8))+'</td><td class="mono">'+projShort(t.projectPath)+'</td><td class="mono">'+esc(t.targetFile||'—')+'</td><td class="mono" title="'+esc(t.createdAt)+'">'+reltime(t.createdAt)+'</td></tr>').join('')
    }
    document.getElementById('refresh-ts').textContent = 'updated '+new Date().toLocaleTimeString()
  } catch(e) {
    document.getElementById('refresh-ts').textContent = 'error: '+esc(e.message)
  }
}

// SSE for push updates; falls back silently to polling if unavailable
try {
  const es = new EventSource('/api/events')
  es.onmessage = e => { const d = JSON.parse(e.data); if (d.type !== 'connected') refresh() }
  es.onerror = () => es.close()
} catch(_) {}

refresh()
setInterval(refresh, 2000)
</script>
</body>
</html>`
