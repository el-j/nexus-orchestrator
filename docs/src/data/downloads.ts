/**
 * Static content for the Downloads page: per-platform release assets,
 * system requirements and package-manager snippets.
 */

export const DESKTOP_ASSETS = [
  {
    icon: '🍎',
    platform: 'macOS',
    arch: 'Apple Silicon (M1/M2/M3/M4)',
    ext: '.zip',
    osKey: 'mac-arm',
    file: 'nexus-orchestrator-desktop-darwin-arm64.zip',
  },
  {
    icon: '🍎',
    platform: 'macOS',
    arch: 'Intel (x86_64)',
    ext: '.zip',
    osKey: 'mac-intel',
    file: 'nexus-orchestrator-desktop-darwin-amd64.zip',
  },
  {
    icon: '🪟',
    platform: 'Windows',
    arch: '64-bit (x86_64)',
    ext: '.zip',
    osKey: 'win',
    file: 'nexus-orchestrator-desktop-windows-amd64.zip',
  },
  {
    icon: '🐧',
    platform: 'Linux',
    arch: '64-bit (x86_64)',
    ext: '.tar.gz',
    osKey: 'linux',
    file: 'nexus-orchestrator-desktop-linux-amd64.tar.gz',
  },
];

export const CLI_ASSETS = [
  {
    icon: '🍎',
    platform: 'macOS',
    arch: 'Apple Silicon (arm64)',
    ext: '.tar.gz',
    osKey: 'mac-arm',
    file: 'nexus-orchestrator-darwin-arm64.tar.gz',
  },
  {
    icon: '🍎',
    platform: 'macOS',
    arch: 'Intel (amd64)',
    ext: '.tar.gz',
    osKey: 'mac-intel',
    file: 'nexus-orchestrator-darwin-amd64.tar.gz',
  },
  {
    icon: '🪟',
    platform: 'Windows',
    arch: '64-bit (amd64)',
    ext: '.zip',
    osKey: 'win',
    file: 'nexus-orchestrator-windows-amd64.zip',
  },
  {
    icon: '🐧',
    platform: 'Linux',
    arch: '64-bit (amd64)',
    ext: '.tar.gz',
    osKey: 'linux',
    file: 'nexus-orchestrator-linux-amd64.tar.gz',
  },
  {
    icon: '🐧',
    platform: 'Linux',
    arch: 'ARM64',
    ext: '.tar.gz',
    osKey: 'linux-arm',
    file: 'nexus-orchestrator-linux-arm64.tar.gz',
  },
];

export const requirements = [
  {
    icon: '🍎',
    title: 'macOS',
    items: [
      'macOS 11 (Big Sur) or later',
      'Apple Silicon or Intel processor',
      'Native .app bundle included',
    ],
  },
  {
    icon: '🪟',
    title: 'Windows',
    items: [
      'Windows 10 (version 1809) or later',
      '64-bit processor (x86_64 or ARM64)',
      'WebView2 runtime (usually pre-installed)',
    ],
  },
  {
    icon: '🐧',
    title: 'Linux',
    items: [
      'Ubuntu 20.04+ / Debian 11+ / Fedora 36+',
      'x86_64 or ARM64 processor',
      'GTK 3 and WebKit2GTK (Desktop app)',
    ],
  },
  {
    icon: '🔧',
    title: 'Build from Source',
    items: [
      'Go 1.26+ with <code class="text-slate-300">CGO_ENABLED=1</code>',
      'C compiler (gcc / clang) for sqlite3',
      '<a href="https://wails.io/" class="text-violet-400">Wails v2</a> for Desktop builds',
    ],
  },
];

export const packages = [
  {
    icon: '🖥️',
    title: 'Desktop App',
    items: [
      'Full Wails GUI with task dashboard',
      'Embedded HTTP API on port <code class="text-slate-300">63987</code>',
      'Embedded MCP server on port <code class="text-slate-300">63988</code>',
      'Provider auto-discovery &amp; health UI',
      'Per-project session viewer',
    ],
  },
  {
    icon: '⚙️',
    title: 'Daemon (nexus-daemon)',
    items: [
      'Headless background service',
      'HTTP API on port <code class="text-slate-300">63987</code>',
      'MCP server on port <code class="text-slate-300">63988</code>',
      'Ideal for servers &amp; CI environments',
      'Configurable via environment variables',
    ],
  },
  {
    icon: '📟',
    title: 'CLI (nexus-cli)',
    items: [
      'Thin HTTP client — no embedded LLM logic',
      'Submit, list, cancel, and monitor tasks',
      'Connects to daemon at <code class="text-slate-300">127.0.0.1:63987</code>',
      'Scriptable for automation &amp; pipelines',
    ],
  },
  {
    icon: '🔌',
    title: 'VS Code Extension',
    items: [
      'Submit tasks directly from the editor',
      'Task queue tree view sidebar',
      'Status bar with live daemon health',
      'Provider &amp; model picker command',
      'Connects to daemon at <code class="text-slate-300">127.0.0.1:63987</code>',
    ],
  },
];
