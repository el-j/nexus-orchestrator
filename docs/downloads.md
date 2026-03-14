---
layout: default
title: Downloads
nav_order: 2
description: "Download nexusOrchestrator desktop app and CLI tools for macOS, Windows, and Linux"
---

<style>
.dl-hero {
  text-align: center;
  padding: 3rem 0 2rem;
  margin-bottom: 2rem;
  border-bottom: 1px solid var(--border-color);
}
.dl-hero h1 {
  font-size: 2.8rem;
  margin-bottom: 0.5rem;
  background: linear-gradient(135deg, #7c3aed, #a78bfa);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.dl-hero .subtitle {
  font-size: 1.25rem;
  color: var(--grey-dk-000);
  margin-bottom: 1rem;
}
.dl-detect {
  display: inline-block;
  padding: 0.6rem 1.2rem;
  border-radius: 8px;
  background: rgba(124, 58, 237, 0.12);
  border: 1px solid rgba(124, 58, 237, 0.3);
  font-size: 0.95rem;
  margin-top: 0.5rem;
}
.dl-detect strong { color: #a78bfa; }

.dl-section-title {
  font-size: 1.6rem;
  margin: 2.5rem 0 0.5rem;
  padding-bottom: 0.4rem;
  border-bottom: 2px solid rgba(124, 58, 237, 0.3);
}
.dl-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 1.2rem;
  margin: 1.2rem 0 2rem;
}
.dl-card {
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.4rem 1.2rem;
  text-align: center;
  transition: border-color 0.2s, box-shadow 0.2s;
  position: relative;
}
.dl-card:hover {
  border-color: #7c3aed;
  box-shadow: 0 0 12px rgba(124, 58, 237, 0.15);
}
.dl-card.recommended {
  border-color: #7c3aed;
  box-shadow: 0 0 0 2px rgba(124, 58, 237, 0.25);
}
.dl-card.recommended::after {
  content: "Recommended";
  position: absolute;
  top: -10px;
  right: 12px;
  background: #7c3aed;
  color: #fff;
  font-size: 0.7rem;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 4px;
  letter-spacing: 0.03em;
  text-transform: uppercase;
}
.dl-card .dl-icon {
  font-size: 2.2rem;
  display: block;
  margin-bottom: 0.5rem;
}
.dl-card .dl-platform {
  font-weight: 600;
  font-size: 1.05rem;
  margin-bottom: 0.15rem;
}
.dl-card .dl-arch {
  font-size: 0.85rem;
  color: var(--grey-dk-000);
  margin-bottom: 0.75rem;
}
.dl-card .dl-size {
  font-size: 0.78rem;
  color: var(--grey-dk-000);
  margin-bottom: 0.8rem;
}
.dl-btn {
  display: inline-block;
  padding: 0.5rem 1.4rem;
  border-radius: 6px;
  background: #7c3aed;
  color: #fff !important;
  text-decoration: none !important;
  font-weight: 600;
  font-size: 0.9rem;
  transition: background 0.2s;
}
.dl-btn:hover { background: #6d28d9; }
.dl-btn-outline {
  display: inline-block;
  padding: 0.45rem 1.2rem;
  border-radius: 6px;
  border: 1px solid #7c3aed;
  color: #a78bfa !important;
  text-decoration: none !important;
  font-weight: 600;
  font-size: 0.85rem;
  transition: background 0.2s, color 0.2s;
}
.dl-btn-outline:hover { background: rgba(124, 58, 237, 0.1); }
.dl-install-box {
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.5rem;
  margin: 1.2rem 0 2rem;
  text-align: center;
}
.dl-install-box code {
  display: block;
  margin: 0.8rem auto;
  padding: 0.7rem 1rem;
  background: rgba(0,0,0,0.3);
  border-radius: 6px;
  font-size: 0.9rem;
  word-break: break-all;
  max-width: 700px;
}
.dl-info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1.2rem;
  margin: 1.2rem 0 2rem;
}
.dl-info-card {
  border: 1px solid var(--border-color);
  border-radius: 10px;
  padding: 1.3rem;
}
.dl-info-card h4 { margin-top: 0; }
.dl-info-card ul { margin-bottom: 0; padding-left: 1.2rem; }
</style>

<div class="dl-hero">
  <h1>Download nexusOrchestrator</h1>
  <p class="subtitle">Get the desktop app or CLI tools for your platform</p>
  <div class="dl-detect" id="os-detect">
    Detecting your platform…
  </div>
</div>

<!-- ─── Desktop App ────────────────────────────────────── -->

<h2 class="dl-section-title">🖥️ Desktop App</h2>

Full GUI with built-in HTTP API and MCP server. Recommended for most users.

<div class="dl-grid" id="desktop-grid">

  <div class="dl-card" data-os="mac-arm">
    <span class="dl-icon">🍎</span>
    <div class="dl-platform">macOS</div>
    <div class="dl-arch">Apple Silicon (M1/M2/M3/M4)</div>
    <div class="dl-size">~15 MB</div>
    <a class="dl-btn" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-desktop-darwin-arm64.zip">Download .zip</a>
  </div>

  <div class="dl-card" data-os="mac-intel">
    <span class="dl-icon">🍎</span>
    <div class="dl-platform">macOS</div>
    <div class="dl-arch">Intel (x86_64)</div>
    <div class="dl-size">~15 MB</div>
    <a class="dl-btn" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-desktop-darwin-amd64.zip">Download .zip</a>
  </div>

  <div class="dl-card" data-os="win">
    <span class="dl-icon">🪟</span>
    <div class="dl-platform">Windows</div>
    <div class="dl-arch">64-bit (x86_64)</div>
    <div class="dl-size">~15 MB</div>
    <a class="dl-btn" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-desktop-windows-amd64.zip">Download .zip</a>
  </div>

  <div class="dl-card" data-os="linux">
    <span class="dl-icon">🐧</span>
    <div class="dl-platform">Linux</div>
    <div class="dl-arch">64-bit (x86_64)</div>
    <div class="dl-size">~15 MB</div>
    <a class="dl-btn" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-desktop-linux-amd64.tar.gz">Download .tar.gz</a>
  </div>

</div>

<!-- ─── CLI + Daemon ───────────────────────────────────── -->

<h2 class="dl-section-title">⌨️ CLI + Daemon</h2>

Headless daemon and thin CLI client. Ideal for servers, CI pipelines, and scripting.

<div class="dl-grid" id="cli-grid">

  <div class="dl-card" data-os="mac-arm">
    <span class="dl-icon">🍎</span>
    <div class="dl-platform">macOS</div>
    <div class="dl-arch">Apple Silicon (arm64)</div>
    <div class="dl-size">~12 MB</div>
    <a class="dl-btn-outline" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-darwin-arm64.tar.gz">Download .tar.gz</a>
  </div>

  <div class="dl-card" data-os="mac-intel">
    <span class="dl-icon">🍎</span>
    <div class="dl-platform">macOS</div>
    <div class="dl-arch">Intel (amd64)</div>
    <div class="dl-size">~12 MB</div>
    <a class="dl-btn-outline" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-darwin-amd64.tar.gz">Download .tar.gz</a>
  </div>

  <div class="dl-card" data-os="win">
    <span class="dl-icon">🪟</span>
    <div class="dl-platform">Windows</div>
    <div class="dl-arch">64-bit (amd64)</div>
    <div class="dl-size">~12 MB</div>
    <a class="dl-btn-outline" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-windows-amd64.zip">Download .zip</a>
  </div>

  <div class="dl-card" data-os="linux">
    <span class="dl-icon">🐧</span>
    <div class="dl-platform">Linux</div>
    <div class="dl-arch">64-bit (amd64)</div>
    <div class="dl-size">~12 MB</div>
    <a class="dl-btn-outline" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-linux-amd64.tar.gz">Download .tar.gz</a>
  </div>

  <div class="dl-card" data-os="linux-arm">
    <span class="dl-icon">🐧</span>
    <div class="dl-platform">Linux</div>
    <div class="dl-arch">ARM64</div>
    <div class="dl-size">~12 MB</div>
    <a class="dl-btn-outline" href="https://github.com/el-j/nexusOrchestrator/releases/latest/download/nexus-orchestrator-linux-arm64.tar.gz">Download .tar.gz</a>
  </div>

</div>

<!-- ─── Quick Install ──────────────────────────────────── -->

<h2 class="dl-section-title">⚡ Quick Install (CLI)</h2>

<div class="dl-install-box">
  Install the latest CLI + Daemon with a single command:
  <code>curl -sSfL https://raw.githubusercontent.com/el-j/nexusOrchestrator/main/scripts/install.sh | sh</code>
  <p style="font-size:0.82rem; color: var(--grey-dk-000); margin-top: 0.5rem;">
    Detects your OS and architecture automatically. Installs to <code style="display:inline; background:rgba(0,0,0,0.3); padding:2px 6px; border-radius:3px;">~/.local/bin</code>.
  </p>
</div>

<!-- ─── Verify ─────────────────────────────────────────── -->

<h2 class="dl-section-title">🔒 Verify Your Download</h2>

Each release includes `SHA256SUMS.txt` and `SHA256SUMS-desktop.txt` with SHA-256 hashes for every archive.

```sh
# 1. Download the checksums file
curl -sSfLO https://github.com/el-j/nexusOrchestrator/releases/latest/download/SHA256SUMS.txt

# 2. Verify your archive (example for macOS Apple Silicon desktop)
shasum -a 256 -c SHA256SUMS.txt --ignore-missing
# or check a specific file:
shasum -a 256 nexus-orchestrator-darwin-arm64.tar.gz
```

On Linux, use `sha256sum` instead of `shasum -a 256`:

```sh
sha256sum -c SHA256SUMS.txt --ignore-missing
```

On Windows (PowerShell):

```powershell
Get-FileHash .\nexus-orchestrator-windows-amd64.zip -Algorithm SHA256
# Compare the output hash against the value in SHA256SUMS.txt or SHA256SUMS-desktop.txt
```

<h2 class="dl-section-title">🍎 macOS: First Run</h2>

<div class="dl-info-card" style="border-color: #f59e0b; background: rgba(245, 158, 11, 0.06); margin: 1.2rem 0 2rem; padding: 1.5rem;">
  <h4 style="color: #f59e0b; margin-top: 0;">⚠️ "Apple could not verify" — this is expected</h4>
  <p>
    nexusOrchestrator is an open-source app that is not (yet) notarized with Apple.
    macOS Gatekeeper will block it on first launch after downloading from the internet.
    This is normal for community-distributed apps.
  </p>
  <p><strong>Option 1 — Right-click to open (recommended):</strong></p>
  <ol>
    <li>In Finder, right-click (or Control-click) <code>nexusOrchestrator.app</code></li>
    <li>Select <strong>"Open"</strong> from the context menu</li>
    <li>Click <strong>"Open"</strong> in the dialog that appears</li>
    <li>The app will be remembered and open normally from now on</li>
  </ol>
  <p><strong>Option 2 — Terminal (bulk clear):</strong></p>
  <code style="display: block; margin: 0.8rem auto; padding: 0.7rem 1rem; background: rgba(0,0,0,0.3); border-radius: 6px; font-size: 0.9rem; max-width: 700px;">xattr -dr com.apple.quarantine /path/to/nexusOrchestrator.app</code>
  <p style="font-size: 0.85rem; color: var(--grey-dk-000); margin-bottom: 0;">
    Replace <code>/path/to/</code> with the actual path (e.g. <code>~/Downloads/</code>).
    This removes the quarantine flag that macOS sets on files downloaded from the internet.
  </p>
</div>

---

<!-- ─── Requirements ───────────────────────────────────── -->

<h2 class="dl-section-title">📋 System Requirements</h2>

<div class="dl-info-grid">

  <div class="dl-info-card">
    <h4>🍎 macOS</h4>
    <ul>
      <li>macOS 11 (Big Sur) or later</li>
      <li>Apple Silicon or Intel processor</li>
      <li>Desktop app is a native .app bundle</li>
    </ul>
  </div>

  <div class="dl-info-card">
    <h4>🪟 Windows</h4>
    <ul>
      <li>Windows 10 (version 1809) or later</li>
      <li>64-bit processor (x86_64 or ARM64)</li>
      <li>WebView2 runtime (usually pre-installed)</li>
    </ul>
  </div>

  <div class="dl-info-card">
    <h4>🐧 Linux</h4>
    <ul>
      <li>Ubuntu 20.04+ / Debian 11+ / Fedora 36+</li>
      <li>x86_64 or ARM64 processor</li>
      <li>GTK 3 and WebKit2GTK (for Desktop app)</li>
    </ul>
  </div>

  <div class="dl-info-card">
    <h4>🔧 Build from Source</h4>
    <ul>
      <li>Go 1.24+ with <code>CGO_ENABLED=1</code></li>
      <li>C compiler (gcc / clang) for sqlite3</li>
      <li><a href="https://wails.io/">Wails v2</a> for Desktop builds</li>
    </ul>
  </div>

</div>

---

<!-- ─── What's Included ───────────────────────────────── -->

<h2 class="dl-section-title">📦 What's Included</h2>

<div class="dl-info-grid">

  <div class="dl-info-card">
    <h4>Desktop App</h4>
    <ul>
      <li>Full Wails GUI with task dashboard</li>
      <li>Embedded HTTP API on port <code>63987</code></li>
      <li>Embedded MCP server on port <code>63988</code></li>
      <li>Provider auto-discovery &amp; health UI</li>
      <li>Per-project session viewer</li>
    </ul>
  </div>

  <div class="dl-info-card">
    <h4>Daemon (<code>nexus-daemon</code>)</h4>
    <ul>
      <li>Headless background service</li>
      <li>HTTP API on port <code>63987</code></li>
      <li>MCP server on port <code>63988</code></li>
      <li>Ideal for servers &amp; CI environments</li>
      <li>Configurable via environment variables</li>
    </ul>
  </div>

  <div class="dl-info-card">
    <h4>CLI (<code>nexus-cli</code>)</h4>
    <ul>
      <li>Thin HTTP client — no embedded LLM logic</li>
      <li>Submit, list, cancel, and monitor tasks</li>
      <li>Connects to daemon at <code>127.0.0.1:63987</code></li>
      <li>Scriptable for automation &amp; pipelines</li>
    </ul>
  </div>

</div>

---

<p style="text-align:center; margin-top:2rem;">
  <a href="/nexusOrchestrator/getting-started" class="dl-btn" style="margin-right:0.5rem;">Getting Started Guide →</a>
  <a href="https://github.com/el-j/nexusOrchestrator" class="dl-btn-outline">View Source on GitHub</a>
</p>

<!-- ─── OS Detection Script ────────────────────────────── -->

<script>
(function() {
  var detect = document.getElementById('os-detect');
  var ua = navigator.userAgent || '';
  var platform = navigator.platform || '';
  var os = 'unknown';
  var arch = 'amd64';
  var label = '';

  if (/Mac/i.test(platform)) {
    os = 'mac';
    label = 'macOS';
    // Heuristic: newer Macs with ARM
    if (/ARM/i.test(ua) || (navigator.userAgentData && navigator.userAgentData.architecture === 'arm')) {
      arch = 'arm64';
      label = 'macOS (Apple Silicon)';
    } else {
      arch = 'amd64';
      label = 'macOS (Intel)';
    }
  } else if (/Win/i.test(platform)) {
    os = 'win';
    label = 'Windows';
    if (/ARM/i.test(ua) || /WOA/i.test(ua)) {
      arch = 'arm64';
      label = 'Windows (ARM — use x64 build)';
    }
  } else if (/Linux/i.test(platform)) {
    os = 'linux';
    label = 'Linux';
    if (/aarch64|arm64/i.test(ua)) {
      arch = 'arm64';
      label = 'Linux (ARM64)';
    }
  }

  if (os === 'unknown') {
    detect.innerHTML = 'Could not auto-detect your platform. Choose your download below.';
    return;
  }

  detect.innerHTML = 'Detected <strong>' + label + '</strong> — your recommended downloads are highlighted below.';

  // Map os+arch to data-os attribute value
  var key = os;
  if (os === 'mac' && arch === 'arm64')  key = 'mac-arm';
  if (os === 'mac' && arch === 'amd64')  key = 'mac-intel';
  if (os === 'linux' && arch === 'arm64') key = 'linux-arm';

  var cards = document.querySelectorAll('.dl-card');
  for (var i = 0; i < cards.length; i++) {
    var cardOs = cards[i].getAttribute('data-os');
    if (cardOs === key) {
      cards[i].classList.add('recommended');
    }
  }
})();
</script>
