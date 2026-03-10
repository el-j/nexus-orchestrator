package httpapi

import (
	"html/template"
	"net/http"
)

var dashboardTmpl = template.Must(template.New("dashboard").Parse(dashboardTemplateHTML))

func (s *Server) handleUI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'unsafe-inline'; script-src 'unsafe-inline'")
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
    /* Task submission form */
    .submit-panel { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; margin-bottom: 1.25rem; overflow: hidden }
    .submit-panel summary { padding: 0.75rem 1rem; font-size: 0.8rem; font-weight: 600;
                            text-transform: uppercase; letter-spacing: .05em; color: var(--muted);
                            cursor: pointer; user-select: none; list-style: none; display: flex; align-items: center; gap: 0.5rem }
    .submit-panel summary::before { content: '▶'; font-size: 0.6rem; transition: transform .2s }
    details[open] > summary::before { transform: rotate(90deg) }
    .submit-form { padding: 1rem; display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem }
    @media(max-width:640px){ .submit-form { grid-template-columns: 1fr } }
    .submit-form .full { grid-column: 1/-1 }
    .field label { display: block; font-size: 0.7rem; color: var(--muted); margin-bottom: 0.3rem; text-transform: uppercase; letter-spacing: .04em }
    .field input, .field select, .field textarea {
      width: 100%; background: var(--surface2); border: 1px solid var(--border);
      border-radius: 6px; color: var(--text); padding: 0.45rem 0.65rem; font-size: 0.8rem; font-family: inherit }
    .field textarea { resize: vertical; min-height: 72px }
    .field input:focus, .field select:focus, .field textarea:focus { outline: 2px solid var(--indigo); outline-offset: -1px }
    .btn { padding: 0.45rem 1rem; border-radius: 6px; border: none; font-size: 0.8rem; font-weight: 600;
           cursor: pointer; transition: opacity .15s }
    .btn:hover { opacity: 0.85 } .btn:disabled { opacity: 0.45; cursor: not-allowed }
    .btn-primary { background: var(--indigo); color: #fff }
    .btn-danger  { background: transparent; border: 1px solid var(--red); color: var(--red); padding: 0.2rem 0.5rem; font-size: 0.7rem }
    .btn-ghost   { background: transparent; border: 1px solid var(--border); color: var(--muted) }
    #submit-status { font-size: 0.75rem; margin-top: 0.5rem; min-height: 1.1em }
    .ok  { color: var(--green) }
    .err { color: var(--red) }
    /* Main layout */
    .grid2 { display: grid; grid-template-columns: 300px 1fr; gap: 1.25rem }
    @media(max-width:768px){ .grid2 { grid-template-columns: 1fr } }
    .panel { background: var(--surface); border: 1px solid var(--border); border-radius: 8px; overflow: hidden }
    .panel-header { padding: 0.75rem 1rem; border-bottom: 1px solid var(--border);
                    display: flex; align-items: center; justify-content: space-between;
                    font-size: 0.8rem; font-weight: 600; text-transform: uppercase; letter-spacing: .05em; color: var(--muted) }
    /* Provider cards */
    .provider-card { padding: 0.65rem 1rem; border-bottom: 1px solid var(--border) }
    .provider-card-row { display: flex; align-items: center; gap: 0.5rem }
    .provider-card-row .name { flex: 1; font-size: 0.85rem }
    .dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0 }
    .dot-green { background: var(--green); box-shadow: 0 0 6px var(--green); animation: pulse 2s infinite }
    .dot-red   { background: var(--red) }
    @keyframes pulse { 0%,100%{opacity:1}50%{opacity:.4} }
    .model-pills { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-top: 0.45rem }
    .pill { background: var(--surface2); border: 1px solid var(--border); border-radius: 9999px;
            font-size: 0.65rem; padding: 0.1rem 0.45rem; color: var(--muted); font-family: monospace }
    .pill-active { border-color: var(--indigo); color: var(--indigo) }
    /* Table */
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
    .badge-toolarge   { background: #451a03; color: #fdba74 }
    .badge-noprovider { background: #2d1561; color: #c4b5fd }
    .mono { font-family: monospace; font-size: 0.75rem; color: var(--muted) }
    .empty { padding: 2rem; text-align: center; color: var(--muted); font-size: 0.85rem }
    #refresh-ts { font-size:0.7rem; color:var(--muted); margin-left:auto }
    /* Modal */
    .modal-backdrop { display: none; position: fixed; inset: 0; background: rgba(0,0,0,.65);
                      z-index: 100; align-items: center; justify-content: center }
    .modal-backdrop.open { display: flex }
    .modal { background: var(--surface); border: 1px solid var(--border); border-radius: 10px;
             width: min(480px, 95vw); padding: 1.25rem; display: flex; flex-direction: column; gap: 0.75rem }
    .modal h2 { font-size: 0.9rem; font-weight: 700 }
    .modal-actions { display: flex; justify-content: flex-end; gap: 0.5rem; margin-top: 0.25rem }
    #modal-err { font-size: 0.75rem; color: var(--red); min-height: 1em }
    .hidden { display: none !important }
  </style>
</head>
<body>
<header>
  <h1>nexusOrchestrator</h1>
  <span class="subtitle">Live Queue Dashboard</span>
  <span id="refresh-ts">–</span>
</header>

<!-- Add-Provider Modal -->
<div id="modal" class="modal-backdrop" role="dialog" aria-modal="true" aria-labelledby="modal-title">
  <div class="modal">
    <h2 id="modal-title">Add Provider</h2>
    <div class="field">
      <label>Name</label>
      <input id="m-name" type="text" placeholder="e.g. my-lmstudio" autocomplete="off">
    </div>
    <div class="field">
      <label>Kind</label>
      <select id="m-kind">
        <option value="lmstudio">LM Studio</option>
        <option value="ollama">Ollama</option>
        <option value="openaicompat">OpenAI-Compatible</option>
        <option value="anthropic">Anthropic</option>
      </select>
    </div>
    <div class="field" id="m-baseurl-row">
      <label>Base URL</label>
      <input id="m-baseurl" type="text" placeholder="http://127.0.0.1:1234">
    </div>
    <div class="field hidden" id="m-apikey-row">
      <label>API Key</label>
      <input id="m-apikey" type="password" placeholder="sk-…">
    </div>
    <div class="field">
      <label>Default Model (optional)</label>
      <input id="m-model" type="text" placeholder="e.g. llama3.2:3b">
    </div>
    <div id="modal-err"></div>
    <div class="modal-actions">
      <button class="btn btn-ghost" id="modal-cancel">Cancel</button>
      <button class="btn btn-primary" id="modal-submit">Add</button>
    </div>
  </div>
</div>

<div class="container">
  <div class="stats">
    <div class="stat-card"><div class="label">Queued</div><div class="value" id="stat-queued">–</div></div>
    <div class="stat-card"><div class="label">Processing</div><div class="value" id="stat-processing">–</div></div>
    <div class="stat-card"><div class="label">Providers Online</div><div class="value" id="stat-providers">–</div></div>
    <div class="stat-card"><div class="label">Total in View</div><div class="value" id="stat-total">–</div></div>
  </div>

  <!-- Task Submit Form -->
  <details class="submit-panel">
    <summary>Submit Task</summary>
    <div class="submit-form">
      <div class="field">
        <label>Project Path</label>
        <input id="f-project" type="text" placeholder="/path/to/project">
      </div>
      <div class="field">
        <label>Target File</label>
        <input id="f-target" type="text" placeholder="src/main.go">
      </div>
      <div class="field full">
        <label>Instruction</label>
        <textarea id="f-instruction" placeholder="Describe what to do…"></textarea>
      </div>
      <div class="field">
        <label>Context Files (comma-separated)</label>
        <input id="f-context" type="text" placeholder="README.md, go.mod">
      </div>
      <div class="field">
        <label>Provider Hint</label>
        <select id="f-provider-hint"><option value="">— any —</option></select>
      </div>
      <div class="field full">
        <label>Model ID</label>
        <select id="f-model"><option value="">— default —</option></select>
      </div>
      <div class="field full">
        <button class="btn btn-primary" id="f-submit">Submit Task</button>
        <div id="submit-status"></div>
      </div>
    </div>
  </details>

  <div class="grid2">
    <div class="panel">
      <div class="panel-header">
        <span>LLM Providers</span>
        <button class="btn btn-primary" id="add-provider-btn" style="font-size:0.7rem;padding:0.2rem 0.65rem">+ Add</button>
      </div>
      <div id="providers-list"><div class="empty">Loading…</div></div>
    </div>
    <div class="panel">
      <div class="panel-header">Task Queue</div>
      <table>
        <thead><tr><th>Status</th><th>ID</th><th>Project</th><th>Target File</th><th>Model</th><th>Submitted</th></tr></thead>
        <tbody id="queue-body"><tr><td colspan="6" class="empty">Loading…</td></tr></tbody>
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
  const map = {
    QUEUED:'badge-queued', PROCESSING:'badge-processing', COMPLETED:'badge-completed',
    FAILED:'badge-failed', CANCELLED:'badge-cancelled',
    TOO_LARGE:'badge-toolarge', NO_PROVIDER:'badge-noprovider'
  }
  return '<span class="badge '+(map[s]||'badge-queued')+'">'+esc(s)+'</span>'
}
function projShort(p) { return esc((p||'').split('/').slice(-2).join('/')) }

// Model cache: { providerName -> string[] }
const modelCache = {}

async function loadModels(name) {
  if (modelCache[name]) return modelCache[name]
  try {
    const r = await fetch('/api/providers/'+encodeURIComponent(name)+'/models')
    if (!r.ok) return []
    modelCache[name] = await r.json()
    return modelCache[name]
  } catch { return [] }
}

async function removeProvider(name) {
  if (!confirm('Remove provider "'+name+'"?')) return
  try {
    const r = await fetch('/api/providers/'+encodeURIComponent(name), {method:'DELETE'})
    if (!r.ok) { alert('Failed to remove: '+(await r.text())); return }
    delete modelCache[name]
    await refresh()
  } catch(e) { alert('Error: '+e.message) }
}

async function populateModelSelects(providers) {
  const hint = document.getElementById('f-provider-hint')
  const model = document.getElementById('f-model')
  const prevHint = hint.value
  const prevModel = model.value
  // rebuild provider-hint options (online providers only)
  while (hint.options.length > 1) hint.remove(1)
  const online = providers.filter(p=>p.active)
  online.forEach(p => {
    const o = document.createElement('option')
    o.value = p.name; o.textContent = p.name
    hint.appendChild(o)
  })
  if (prevHint) hint.value = prevHint
  // rebuild model options grouped by provider
  while (model.options.length > 1) model.remove(1)
  for (const p of online) {
    const models = await loadModels(p.name)
    if (models.length === 0) continue
    const grp = document.createElement('optgroup')
    grp.label = p.name
    models.forEach(m => {
      const o = document.createElement('option')
      o.value = m; o.textContent = m
      grp.appendChild(o)
    })
    model.appendChild(grp)
  }
  if (prevModel) model.value = prevModel
}

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

    // Render provider cards
    if (providers.length === 0) {
      document.getElementById('providers-list').innerHTML = '<div class="empty">No providers configured. Click + Add.</div>'
    } else {
      const cards = await Promise.all(providers.map(async prov => {
        const models = await loadModels(prov.name)
        const pills = models.map(m => {
          const active = prov.activeModel && m === prov.activeModel
          return '<span class="pill'+(active?' pill-active':'')+'" title="'+esc(m)+'">'+esc(m)+'</span>'
        }).join('')
        return '<div class="provider-card">' +
          '<div class="provider-card-row">' +
            '<span class="dot '+(prov.active?'dot-green':'dot-red')+'"></span>' +
            '<span class="name">'+esc(prov.name)+'</span>' +
            (prov.activeModel ? '<span class="pill pill-active">'+esc(prov.activeModel)+'</span>' : '') +
            '<button class="btn btn-danger" onclick="removeProvider(\''+esc(prov.name)+'\')">×</button>' +
          '</div>' +
          (models.length > 0 ? '<div class="model-pills">'+pills+'</div>' : '') +
        '</div>'
      }))
      document.getElementById('providers-list').innerHTML = cards.join('')
    }

    // Update model selects
    await populateModelSelects(providers)

    // Render task table
    const tbody = document.getElementById('queue-body')
    if (tasks.length === 0) {
      tbody.innerHTML = '<tr><td colspan="6" class="empty">Queue is empty — submit tasks via the form above or POST /api/tasks</td></tr>'
    } else {
      tbody.innerHTML = tasks.map(t=>'<tr>' +
        '<td>'+badge(t.status)+'</td>' +
        '<td class="mono" title="'+esc(t.id)+'">'+esc(t.id.slice(0,8))+'</td>' +
        '<td class="mono">'+projShort(t.projectPath)+'</td>' +
        '<td class="mono">'+esc(t.targetFile||'—')+'</td>' +
        '<td class="mono">'+esc(t.modelId||'—')+'</td>' +
        '<td class="mono" title="'+esc(t.createdAt)+'">'+reltime(t.createdAt)+'</td>' +
      '</tr>').join('')
    }
    document.getElementById('refresh-ts').textContent = 'updated '+new Date().toLocaleTimeString()
  } catch(e) {
    document.getElementById('refresh-ts').textContent = 'error: '+esc(e.message)
  }
}

// --- Task submit ---
document.getElementById('f-submit').addEventListener('click', async () => {
  const btn = document.getElementById('f-submit')
  const status = document.getElementById('submit-status')
  const projectPath = document.getElementById('f-project').value.trim()
  const targetFile  = document.getElementById('f-target').value.trim()
  const instruction = document.getElementById('f-instruction').value.trim()
  if (!projectPath || !instruction) {
    status.innerHTML = '<span class="err">Project path and instruction are required.</span>'
    return
  }
  const contextRaw = document.getElementById('f-context').value.trim()
  const contextFiles = contextRaw ? contextRaw.split(',').map(s=>s.trim()).filter(Boolean) : []
  const modelId      = document.getElementById('f-model').value || undefined
  const providerHint = document.getElementById('f-provider-hint').value || undefined
  const body = {projectPath, targetFile, instruction, contextFiles}
  if (modelId)      body.modelId = modelId
  if (providerHint) body.providerHint = providerHint
  btn.disabled = true
  status.innerHTML = '<span class="ok">Submitting…</span>'
  try {
    const r = await fetch('/api/tasks', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(body)})
    if (!r.ok) { const t=await r.text(); throw new Error(t) }
    const {id} = await r.json()
    status.innerHTML = '<span class="ok">Submitted: '+esc(id)+'</span>'
    await refresh()
  } catch(e) {
    status.innerHTML = '<span class="err">Error: '+esc(e.message)+'</span>'
  } finally {
    btn.disabled = false
  }
})

// --- Add-Provider Modal ---
const modal = document.getElementById('modal')
const kindSel = document.getElementById('m-kind')

function syncModalFields() {
  const k = kindSel.value
  document.getElementById('m-baseurl-row').classList.toggle('hidden', k === 'anthropic')
  document.getElementById('m-apikey-row').classList.toggle('hidden', k !== 'openaicompat' && k !== 'anthropic')
}
kindSel.addEventListener('change', syncModalFields)
syncModalFields()

document.getElementById('add-provider-btn').addEventListener('click', () => {
  document.getElementById('modal-err').textContent = ''
  document.getElementById('m-name').value = ''
  document.getElementById('m-baseurl').value = ''
  document.getElementById('m-apikey').value = ''
  document.getElementById('m-model').value = ''
  kindSel.value = 'lmstudio'
  syncModalFields()
  modal.classList.add('open')
  document.getElementById('m-name').focus()
})
document.getElementById('modal-cancel').addEventListener('click', () => modal.classList.remove('open'))
modal.addEventListener('click', e => { if (e.target === modal) modal.classList.remove('open') })

document.getElementById('modal-submit').addEventListener('click', async () => {
  const errEl = document.getElementById('modal-err')
  const name = document.getElementById('m-name').value.trim()
  if (!name) { errEl.textContent = 'Name is required.'; return }
  const cfg = {
    name,
    kind:    kindSel.value,
    baseUrl: document.getElementById('m-baseurl').value.trim() || undefined,
    apiKey:  document.getElementById('m-apikey').value.trim()  || undefined,
    model:   document.getElementById('m-model').value.trim()   || undefined,
  }
  const btn = document.getElementById('modal-submit')
  btn.disabled = true
  errEl.textContent = ''
  try {
    const r = await fetch('/api/providers', {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(cfg)})
    if (!r.ok) { const t=await r.text(); throw new Error(t) }
    modal.classList.remove('open')
    await refresh()
  } catch(e) {
    errEl.textContent = 'Error: '+e.message
  } finally {
    btn.disabled = false
  }
})

// --- SSE push ---
try {
  const es = new EventSource('/api/events')
  es.onmessage = e => { const d = JSON.parse(e.data); if (d.type !== 'connected') refresh() }
  es.onerror = () => es.close()
} catch(_) {}

refresh()
setInterval(refresh, 3000)
</script>
</body>
</html>`
