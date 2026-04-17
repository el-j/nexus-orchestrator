<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header -->
    <header
      class="flex items-center justify-between px-5 py-3 border-b border-white/5 bg-[#0a0a10] flex-shrink-0"
    >
      <div>
        <h1 class="text-sm font-bold text-white">Settings</h1>
        <p class="text-xs text-slate-500">
          Provider connections, queue limits, and server addresses
        </p>
      </div>
    </header>

    <!-- Content -->
    <div class="flex-1 overflow-auto p-5 space-y-6">
      <!-- Config / token error -->
      <div
        v-if="configError || tokenError"
        class="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-xl p-4"
      >
        {{ configError ?? tokenError }}
      </div>

      <!-- 1. Queue Settings -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Queue Settings
        </h2>
        <div class="rounded-xl border border-white/5 bg-white/[0.02] p-4">
          <div class="flex items-center gap-4 mb-2">
            <span class="text-sm text-slate-300">Queue Cap</span>
            <span class="text-xl font-bold text-violet-300 tabular-nums">{{
              runtimeConfig?.queueCap ?? 50
            }}</span>
          </div>
          <p class="text-xs text-slate-500">
            Maximum tasks held in queue. Change with the
            <code class="font-mono text-slate-400 bg-white/5 px-1 py-0.5 rounded"
              >NEXUS_QUEUE_CAP</code
            >
            environment variable and daemon restart.
          </p>
        </div>
      </section>

      <!-- 4. Security / Token Management -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Security</h2>
        <div class="space-y-2">
          <!-- API Token row -->
          <div
            class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02]"
          >
            <div>
              <span class="text-sm font-medium text-white">HTTP API Token</span>
              <p class="text-xs text-slate-500 mt-0.5">
                Protects all <code class="font-mono bg-white/5 px-1 rounded">/api/*</code> endpoints
              </p>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span
                :class="
                  runtimeConfig?.apiTokenEnabled
                    ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
                    : 'text-slate-500 bg-white/[0.02] border-white/5'
                "
                class="text-[10px] px-2 py-0.5 rounded border font-medium"
              >
                {{ runtimeConfig?.apiTokenEnabled ? 'Protected' : 'No auth' }}
              </span>
              <button
                v-if="!runtimeConfig?.apiTokenEnabled"
                class="text-[10px] px-2.5 py-1 rounded border border-violet-500/30 text-violet-400 hover:bg-violet-500/10 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                :disabled="tokenOperation !== null"
                @click="rotateToken('api')"
              >
                {{ isTokenActionBusy('api', 'rotate') ? 'Working…' : 'Generate Token' }}
              </button>
              <template v-else>
                <button
                  class="text-[10px] px-2.5 py-1 rounded border border-violet-500/30 text-violet-400 hover:bg-violet-500/10 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tokenOperation !== null"
                  @click="rotateToken('api')"
                >
                  {{ isTokenActionBusy('api', 'rotate') ? 'Working…' : 'Rotate' }}
                </button>
                <button
                  class="text-[10px] px-2.5 py-1 rounded border border-white/10 text-slate-400 hover:text-red-400 hover:border-red-500/30 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tokenOperation !== null"
                  @click="disableToken('api')"
                >
                  {{ isTokenActionBusy('api', 'disable') ? 'Disabling…' : 'Disable' }}
                </button>
              </template>
            </div>
          </div>
          <!-- MCP Token row -->
          <div
            class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02]"
          >
            <div>
              <span class="text-sm font-medium text-white">MCP Server Token</span>
              <p class="text-xs text-slate-500 mt-0.5">
                Protects the MCP endpoint at
                <code class="font-mono bg-white/5 px-1 rounded">/mcp</code>
              </p>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span
                :class="
                  runtimeConfig?.mcpTokenEnabled
                    ? 'text-emerald-400 bg-emerald-500/10 border-emerald-500/20'
                    : 'text-slate-500 bg-white/[0.02] border-white/5'
                "
                class="text-[10px] px-2 py-0.5 rounded border font-medium"
              >
                {{ runtimeConfig?.mcpTokenEnabled ? 'Protected' : 'No auth' }}
              </span>
              <button
                v-if="!runtimeConfig?.mcpTokenEnabled"
                class="text-[10px] px-2.5 py-1 rounded border border-violet-500/30 text-violet-400 hover:bg-violet-500/10 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                :disabled="tokenOperation !== null"
                @click="rotateToken('mcp')"
              >
                {{ isTokenActionBusy('mcp', 'rotate') ? 'Working…' : 'Generate Token' }}
              </button>
              <template v-else>
                <button
                  class="text-[10px] px-2.5 py-1 rounded border border-violet-500/30 text-violet-400 hover:bg-violet-500/10 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tokenOperation !== null"
                  @click="rotateToken('mcp')"
                >
                  {{ isTokenActionBusy('mcp', 'rotate') ? 'Working…' : 'Rotate' }}
                </button>
                <button
                  class="text-[10px] px-2.5 py-1 rounded border border-white/10 text-slate-400 hover:text-red-400 hover:border-red-500/30 transition-colors disabled:opacity-60 disabled:cursor-not-allowed"
                  :disabled="tokenOperation !== null"
                  @click="disableToken('mcp')"
                >
                  {{ isTokenActionBusy('mcp', 'disable') ? 'Disabling…' : 'Disable' }}
                </button>
              </template>
            </div>
          </div>
          <!-- One-time token display -->
          <div
            v-if="newToken"
            class="flex items-center gap-2 px-4 py-3 rounded-xl border border-emerald-500/20 bg-emerald-500/5"
          >
            <div class="flex-1 min-w-0">
              <p class="text-xs text-emerald-400 mb-1">
                {{ newToken.kind === 'api' ? 'API' : 'MCP' }} token — copy now, it won't be shown
                again
              </p>
              <code class="text-xs font-mono text-white break-all">{{ newToken.value }}</code>
            </div>
            <button
              class="text-[10px] flex-shrink-0 px-2.5 py-1 rounded border transition-colors"
              :class="
                tokenCopied
                  ? 'border-emerald-500/30 text-emerald-400 bg-emerald-500/5'
                  : 'border-white/10 text-slate-400 hover:text-emerald-400'
              "
              @click="copyToken"
            >
              {{ tokenCopied ? '✓ Copied' : 'Copy' }}
            </button>
            <button
              class="text-[10px] flex-shrink-0 px-2 py-1 rounded border border-white/10 text-slate-400 hover:text-white transition-colors"
              @click="newToken = null"
            >
              ✕
            </button>
          </div>
        </div>
      </section>

      <!-- 3. Server Addresses -->
      <section>
        <h2 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">
          Server Addresses
        </h2>
        <div class="space-y-2">
          <div
            v-for="addr in serverAddresses"
            :key="addr.label"
            class="flex items-center justify-between gap-3 px-4 py-3 rounded-xl border border-white/5 bg-white/[0.02]"
          >
            <div class="flex items-center gap-3 min-w-0">
              <span class="text-xs font-medium text-slate-400 flex-shrink-0 w-24">{{
                addr.label
              }}</span>
              <span class="text-xs font-mono text-slate-300 truncate">{{ addr.url }}</span>
            </div>
            <button
              class="text-[10px] flex-shrink-0 px-2.5 py-1 rounded border transition-colors"
              :class="
                copied === addr.url
                  ? 'border-emerald-500/30 text-emerald-400 bg-emerald-500/5'
                  : 'border-white/10 text-slate-400 hover:text-violet-400 hover:border-violet-500/30'
              "
              @click="copyToClipboard(addr.url)"
            >
              {{ copied === addr.url ? '✓ Copied' : 'Copy' }}
            </button>
          </div>
        </div>
        <p class="text-xs text-slate-600 mt-2">
          Changes require daemon restart. Configure via
          <code class="font-mono text-slate-500">NEXUS_LISTEN_ADDR</code> /
          <code class="font-mono text-slate-500">NEXUS_MCP_ADDR</code>.
        </p>
      </section>
    </div>

    <!-- Confirm dialog -->
    <AppConfirmDialog
      :open="confirmDialog.open"
      :title="confirmDialog.title"
      :message="confirmDialog.message"
      :danger="true"
      @confirm="
        confirmDialog.onConfirm();
        confirmDialog.open = false;
      "
      @cancel="confirmDialog.open = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, reactive } from 'vue';
import type { RuntimeConfig } from '../types/domain';
import { getRuntimeConfig, updateRuntimeConfig } from '../types/wails';
import { resolveServerUrl } from '../composables/useServerUrl';
import AppConfirmDialog from '../components/AppConfirmDialog.vue';

// ── Confirm dialog state ──────────────────────────────────────────────────────

const confirmDialog = reactive({
  open: false,
  title: '',
  message: '',
  onConfirm: () => {},
});

function showConfirm(title: string, message: string, onConfirm: () => void) {
  confirmDialog.title = title;
  confirmDialog.message = message;
  confirmDialog.onConfirm = onConfirm;
  confirmDialog.open = true;
}

// ── Runtime Config ────────────────────────────────────────────────────────────

const runtimeConfig = ref<RuntimeConfig | null>(null);
const configError = ref<string | null>(null);

async function loadConfig() {
  try {
    runtimeConfig.value = await getRuntimeConfig();
    configError.value = null;
  } catch (e) {
    configError.value = e instanceof Error ? e.message : 'Failed to load configuration';
  }
}

// ── Security / Token Management ───────────────────────────────────────────────

const newToken = ref<{ kind: 'api' | 'mcp'; value: string } | null>(null);
const tokenError = ref<string | null>(null);
const tokenOperation = ref<{ kind: 'api' | 'mcp'; action: 'rotate' | 'disable' } | null>(null);

function isTokenActionBusy(kind: 'api' | 'mcp', action: 'rotate' | 'disable'): boolean {
  return tokenOperation.value?.kind === kind && tokenOperation.value?.action === action;
}

async function rotateToken(kind: 'api' | 'mcp') {
  if (tokenOperation.value) return;
  newToken.value = null;
  tokenError.value = null;
  const update = kind === 'api' ? { rotateApiToken: true } : { rotateMcpToken: true };
  tokenOperation.value = { kind, action: 'rotate' };
  try {
    const cfg = await updateRuntimeConfig(update);
    const val = kind === 'api' ? cfg.apiToken : cfg.mcpToken;
    if (val) newToken.value = { kind, value: val };
    await loadConfig();
  } catch (e) {
    tokenError.value = e instanceof Error ? e.message : 'Failed to rotate token';
  } finally {
    tokenOperation.value = null;
  }
}

async function disableToken(kind: 'api' | 'mcp') {
  showConfirm(
    `Disable ${kind === 'api' ? 'HTTP API' : 'MCP'} token`,
    `This will remove token authentication from the ${kind === 'api' ? 'HTTP API' : 'MCP'} endpoint.`,
    async () => {
      if (tokenOperation.value) return;
      const update = kind === 'api' ? { apiToken: '' } : { mcpToken: '' };
      tokenOperation.value = { kind, action: 'disable' };
      try {
        await updateRuntimeConfig(update);
        newToken.value = null;
        tokenError.value = null;
        await loadConfig();
      } catch (e) {
        tokenError.value = e instanceof Error ? e.message : 'Failed to disable token';
      } finally {
        tokenOperation.value = null;
      }
    },
  );
}

const tokenCopied = ref(false);
let tokenCopyTimer: ReturnType<typeof setTimeout> | null = null;
async function copyToken() {
  if (!newToken.value) return;
  try {
    await navigator.clipboard.writeText(newToken.value.value);
    tokenCopied.value = true;
    if (tokenCopyTimer) clearTimeout(tokenCopyTimer);
    tokenCopyTimer = setTimeout(() => {
      tokenCopied.value = false;
    }, 2000);
  } catch {
    /* clipboard unavailable */
  }
}

// ── Server Addresses ──────────────────────────────────────────────────────────

const resolvedApiUrl = ref('http://127.0.0.1:63987');
const resolvedMcpUrl = ref('http://127.0.0.1:63988/mcp');

onMounted(async () => {
  await loadConfig();
  try {
    const base = await resolveServerUrl();
    resolvedApiUrl.value = base;
    resolvedMcpUrl.value = base.replace(/:\d+/, ':63988') + '/mcp';
  } catch {
    /* keep defaults */
  }
});

onUnmounted(() => {
  if (tokenCopyTimer) clearTimeout(tokenCopyTimer);
  if (copyTimer) clearTimeout(copyTimer);
});

const serverAddresses = computed(() => [
  { label: 'HTTP API', url: resolvedApiUrl.value },
  { label: 'MCP Server', url: resolvedMcpUrl.value },
]);

const copied = ref<string | null>(null);
let copyTimer: ReturnType<typeof setTimeout> | null = null;

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    copied.value = text;
    if (copyTimer) clearTimeout(copyTimer);
    copyTimer = setTimeout(() => {
      copied.value = null;
    }, 2000);
  } catch {
    /* clipboard unavailable */
  }
}
</script>
