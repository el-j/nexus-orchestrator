<template>
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-16">
    <!-- Hero -->
    <div class="text-center mb-16 reveal">
      <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full border border-violet-500/30 bg-violet-500/5 text-sm text-violet-300 mb-6">
        <i class="pi pi-download text-xs"></i>
        Latest release always available on GitHub
      </div>
      <h1 class="text-4xl sm:text-5xl font-black mb-4">
        Download <span class="gradient-text">nexus-orchestrator</span>
      </h1>
      <p class="text-xl text-slate-400 mb-6">Get the desktop app or CLI tools for your platform</p>

      <!-- OS Detection badge -->
      <div class="inline-flex items-center gap-2 px-4 py-2 rounded-xl bg-[#0d0d14] border border-white/10 text-sm">
        <span class="text-slate-400">Detected:</span>
        <span class="font-semibold text-violet-300">{{ detectedOS }}</span>
      </div>
    </div>

    <!-- Desktop App Section -->
    <section class="mb-16">
      <div class="flex items-center gap-3 mb-6 reveal">
        <span class="text-2xl">🖥️</span>
        <h2 class="text-2xl font-black">Desktop App</h2>
        <span class="px-2 py-0.5 text-xs rounded-full bg-violet-600/20 border border-violet-500/30 text-violet-300">Recommended</span>
      </div>
      <p class="text-slate-500 mb-8 reveal">Full GUI with built-in HTTP API and MCP server. Best for most users.</p>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div
          v-for="card in desktopCards"
          :key="card.platform + card.arch"
          :class="[
            'reveal rounded-xl border p-6 text-center transition-all group',
            isRecommended(card.osKey)
              ? 'border-violet-500/60 bg-violet-600/5 glow-purple'
              : 'border-white/8 bg-[#0d0d14] hover:border-violet-500/30'
          ]"
        >
          <div class="text-3xl mb-3">{{ card.icon }}</div>
          <div v-if="isRecommended(card.osKey)" class="mb-2">
            <span class="px-2 py-0.5 text-xs rounded-full bg-violet-600 text-white font-bold">Recommended</span>
          </div>
          <div class="font-bold text-white mb-1">{{ card.platform }}</div>
          <div class="text-xs text-slate-500 mb-1">{{ card.arch }}</div>
          <div class="text-xs text-slate-600 mb-4">~15 MB</div>
          <a
            :href="card.url"
            class="inline-flex items-center justify-center gap-1.5 w-full py-2 rounded-lg bg-violet-600 hover:bg-violet-500 text-white text-sm font-semibold transition-colors"
          >
            <i class="pi pi-download text-xs"></i>
            {{ card.ext }}
          </a>
        </div>
      </div>
    </section>

    <!-- CLI + Daemon Section -->
    <section class="mb-16">
      <div class="flex items-center gap-3 mb-6 reveal">
        <span class="text-2xl">⌨️</span>
        <h2 class="text-2xl font-black">CLI + Daemon</h2>
      </div>
      <p class="text-slate-500 mb-8 reveal">Headless daemon and thin CLI client. Ideal for servers, CI pipelines, and scripting.</p>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
        <div
          v-for="card in cliCards"
          :key="card.platform + card.arch"
          :class="[
            'reveal rounded-xl border p-5 text-center transition-all',
            isRecommended(card.osKey)
              ? 'border-violet-500/40 bg-violet-600/5'
              : 'border-white/8 bg-[#0d0d14] hover:border-violet-500/20'
          ]"
        >
          <div class="text-2xl mb-2">{{ card.icon }}</div>
          <div class="font-semibold text-white text-sm mb-1">{{ card.platform }}</div>
          <div class="text-xs text-slate-500 mb-3">{{ card.arch }}</div>
          <a
            :href="card.url"
            class="inline-flex items-center justify-center gap-1 w-full py-1.5 rounded-lg border border-violet-500/40 text-violet-300 hover:bg-violet-600/10 text-xs font-semibold transition-colors"
          >
            <i class="pi pi-download text-xs"></i>
            {{ card.ext }}
          </a>
        </div>
      </div>
    </section>

    <!-- Quick Install -->
    <section class="mb-16 reveal">
      <div class="flex items-center gap-3 mb-6">
        <span class="text-2xl">⚡</span>
        <h2 class="text-2xl font-black">Quick Install</h2>
      </div>
      <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-8 text-center">
        <p class="text-slate-400 mb-4">Install the latest CLI + Daemon with a single command:</p>
        <CodeBlock language="bash" :code="installScript" />
        <p class="text-xs text-slate-600 mt-4">
          Detects your OS and architecture automatically. Installs to <code class="text-slate-400">~/.local/bin</code>.
        </p>
      </div>
    </section>

    <!-- Verify Section -->
    <section class="mb-16 reveal">
      <div class="flex items-center gap-3 mb-6">
        <span class="text-2xl">🔒</span>
        <h2 class="text-2xl font-black">Verify Your Download</h2>
      </div>
      <p class="text-slate-500 mb-6">Each release includes <code class="text-slate-300">SHA256SUMS.txt</code> with SHA-256 hashes for every archive.</p>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xl">🍎</span>
            <h3 class="font-bold text-white">macOS / Linux</h3>
          </div>
          <CodeBlock language="bash" :code="verifyMac" />
        </div>
        <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xl">🐧</span>
            <h3 class="font-bold text-white">Linux (sha256sum)</h3>
          </div>
          <CodeBlock language="bash" :code="verifyLinux" />
        </div>
        <div class="rounded-xl border border-white/8 bg-[#0d0d14] p-6">
          <div class="flex items-center gap-2 mb-3">
            <span class="text-xl">🪟</span>
            <h3 class="font-bold text-white">Windows (PowerShell)</h3>
          </div>
          <CodeBlock language="powershell" :code="verifyWin" />
        </div>
      </div>
    </section>

    <!-- System Requirements -->
    <section class="mb-16 reveal">
      <div class="flex items-center gap-3 mb-6">
        <span class="text-2xl">📋</span>
        <h2 class="text-2xl font-black">System Requirements</h2>
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <div
          v-for="req in requirements"
          :key="req.title"
          class="rounded-xl border border-white/8 bg-[#0d0d14] p-6"
        >
          <div class="text-2xl mb-3">{{ req.icon }}</div>
          <h3 class="font-bold text-white mb-3">{{ req.title }}</h3>
          <ul class="space-y-1.5 list-none p-0">
            <li v-for="item in req.items" :key="item" class="text-xs text-slate-500 flex items-start gap-1.5">
              <i class="pi pi-check text-violet-500 text-xs mt-0.5 flex-shrink-0"></i>
              <span v-html="item"></span>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- What's Included -->
    <section class="mb-16 reveal">
      <div class="flex items-center gap-3 mb-6">
        <span class="text-2xl">📦</span>
        <h2 class="text-2xl font-black">What's Included</h2>
      </div>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div
          v-for="pkg in packages"
          :key="pkg.title"
          class="rounded-xl border border-white/8 bg-[#0d0d14] hover:border-violet-500/20 p-6 transition-all"
        >
          <h3 class="font-bold text-white mb-4 flex items-center gap-2">
            <span class="text-lg">{{ pkg.icon }}</span>
            {{ pkg.title }}
          </h3>
          <ul class="space-y-2 list-none p-0">
            <li v-for="item in pkg.items" :key="item" class="text-sm text-slate-400 flex items-start gap-2">
              <i class="pi pi-check-circle text-emerald-500 text-xs mt-0.5 flex-shrink-0"></i>
              <span v-html="item"></span>
            </li>
          </ul>
        </div>
      </div>
    </section>

    <!-- CTA links -->
    <div class="text-center reveal">
      <RouterLink
        to="/getting-started"
        class="inline-flex items-center gap-2 px-6 py-3 rounded-xl bg-violet-600 hover:bg-violet-500 text-white font-bold transition-all mr-3"
      >
        Getting Started Guide <i class="pi pi-arrow-right text-xs"></i>
      </RouterLink>
      <a
        href="https://github.com/el-j/nexus-orchestrator"
        target="_blank"
        rel="noopener"
        class="inline-flex items-center gap-2 px-6 py-3 rounded-xl border border-white/10 hover:border-violet-500/40 text-slate-300 hover:text-white font-bold transition-all"
      >
        <i class="pi pi-github"></i> View Source on GitHub
      </a>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { RouterLink } from 'vue-router'
import CodeBlock from '../components/CodeBlock.vue'

const detectedOS = ref('Detecting...')
const detectedKey = ref('')

onMounted(() => {
  const ua = navigator.userAgent || ''
  const platform = navigator.platform || ''

  if (/Mac/i.test(platform)) {
    if (/ARM/i.test(ua)) {
      detectedOS.value = 'macOS (Apple Silicon)'
      detectedKey.value = 'mac-arm'
    } else {
      detectedOS.value = 'macOS (Intel)'
      detectedKey.value = 'mac-intel'
    }
  } else if (/Win/i.test(platform)) {
    detectedOS.value = 'Windows 64-bit'
    detectedKey.value = 'win'
  } else if (/Linux/i.test(platform)) {
    if (/aarch64|arm64/i.test(ua)) {
      detectedOS.value = 'Linux ARM64'
      detectedKey.value = 'linux-arm'
    } else {
      detectedOS.value = 'Linux 64-bit'
      detectedKey.value = 'linux'
    }
  } else {
    detectedOS.value = 'Unknown — choose below'
  }
})

function isRecommended(osKey: string) {
  return detectedKey.value && detectedKey.value === osKey
}

const baseURL = 'https://github.com/el-j/nexus-orchestrator/releases/latest/download'

const desktopCards = [
  { icon: '🍎', platform: 'macOS', arch: 'Apple Silicon (M1/M2/M3/M4)', ext: '.tar.gz', osKey: 'mac-arm', url: `${baseURL}/nexus-orchestrator-desktop-darwin-arm64.tar.gz` },
  { icon: '🍎', platform: 'macOS', arch: 'Intel (x86_64)', ext: '.tar.gz', osKey: 'mac-intel', url: `${baseURL}/nexus-orchestrator-desktop-darwin-amd64.tar.gz` },
  { icon: '🪟', platform: 'Windows', arch: '64-bit (x86_64)', ext: '.zip', osKey: 'win', url: `${baseURL}/nexus-orchestrator-desktop-windows-amd64.zip` },
  { icon: '🐧', platform: 'Linux', arch: '64-bit (x86_64)', ext: '.tar.gz', osKey: 'linux', url: `${baseURL}/nexus-orchestrator-desktop-linux-amd64.tar.gz` },
]

const cliCards = [
  { icon: '🍎', platform: 'macOS', arch: 'Apple Silicon (arm64)', ext: '.tar.gz', osKey: 'mac-arm', url: `${baseURL}/nexus-orchestrator-darwin-arm64.tar.gz` },
  { icon: '🍎', platform: 'macOS', arch: 'Intel (amd64)', ext: '.tar.gz', osKey: 'mac-intel', url: `${baseURL}/nexus-orchestrator-darwin-amd64.tar.gz` },
  { icon: '🪟', platform: 'Windows', arch: '64-bit (amd64)', ext: '.zip', osKey: 'win', url: `${baseURL}/nexus-orchestrator-windows-amd64.zip` },
  { icon: '🐧', platform: 'Linux', arch: '64-bit (amd64)', ext: '.tar.gz', osKey: 'linux', url: `${baseURL}/nexus-orchestrator-linux-amd64.tar.gz` },
  { icon: '🐧', platform: 'Linux', arch: 'ARM64', ext: '.tar.gz', osKey: 'linux-arm', url: `${baseURL}/nexus-orchestrator-linux-arm64.tar.gz` },
]

const installScript = `curl -sSfL https://raw.githubusercontent.com/el-j/nexus-orchestrator/main/scripts/install.sh | sh`

const verifyMac = `# Download checksums
curl -sSfLO https://github.com/el-j/nexus-orchestrator/releases/latest/download/SHA256SUMS.txt

# Verify (macOS)
shasum -a 256 -c SHA256SUMS.txt --ignore-missing`

const verifyLinux = `# Download checksums
curl -sSfLO https://github.com/el-j/nexus-orchestrator/releases/latest/download/SHA256SUMS.txt

# Verify (Linux)
sha256sum -c SHA256SUMS.txt --ignore-missing`

const verifyWin = `# PowerShell
Get-FileHash .\\nexus-orchestrator-windows-amd64.zip -Algorithm SHA256
# Compare with SHA256SUMS.txt`

const requirements = [
  {
    icon: '🍎',
    title: 'macOS',
    items: ['macOS 11 (Big Sur) or later', 'Apple Silicon or Intel processor', 'Native .app bundle included'],
  },
  {
    icon: '🪟',
    title: 'Windows',
    items: ['Windows 10 (version 1809) or later', '64-bit processor (x86_64 or ARM64)', 'WebView2 runtime (usually pre-installed)'],
  },
  {
    icon: '🐧',
    title: 'Linux',
    items: ['Ubuntu 20.04+ / Debian 11+ / Fedora 36+', 'x86_64 or ARM64 processor', 'GTK 3 and WebKit2GTK (Desktop app)'],
  },
  {
    icon: '🔧',
    title: 'Build from Source',
    items: ['Go 1.24+ with <code class="text-slate-300">CGO_ENABLED=1</code>', 'C compiler (gcc / clang) for sqlite3', '<a href="https://wails.io/" class="text-violet-400">Wails v2</a> for Desktop builds'],
  },
]

const packages = [
  {
    icon: '🖥️',
    title: 'Desktop App',
    items: [
      'Full Wails GUI with task dashboard',
      'Embedded HTTP API on port <code class="text-slate-300">9999</code>',
      'Embedded MCP server on port <code class="text-slate-300">9998</code>',
      'Provider auto-discovery &amp; health UI',
      'Per-project session viewer',
    ],
  },
  {
    icon: '⚙️',
    title: 'Daemon (nexus-daemon)',
    items: [
      'Headless background service',
      'HTTP API on port <code class="text-slate-300">9999</code>',
      'MCP server on port <code class="text-slate-300">9998</code>',
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
      'Connects to daemon at <code class="text-slate-300">127.0.0.1:9999</code>',
      'Scriptable for automation &amp; pipelines',
    ],
  },
]
</script>
