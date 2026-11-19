<!--
  Token usage of the last 24 hours from the usage ledger: totals, and one row per provider, model and
  role with calls, tokens and cost. Shows why it is empty (no calls yet, or the daemon has no ledger).
  No props.
-->
<template>
  <div class="rounded-xl border border-white/5 bg-white/2 p-4 text-white" data-testid="usage-card">
    <div class="flex items-center justify-between mb-3">
      <h3 class="text-xs font-semibold text-slate-400 uppercase tracking-wider">
        Token usage (24 h)
      </h3>
      <span v-if="report" class="text-xs text-slate-400" data-testid="usage-totals">
        {{ report.totals.calls }} calls · {{ format(report.totals.inputTokens) }} in ·
        {{ format(report.totals.outputTokens) }} out
        <template v-if="report.totals.costUSD"> · ${{ report.totals.costUSD.toFixed(4) }}</template>
      </span>
    </div>
    <p v-if="error" class="text-xs text-red-400" data-testid="usage-error">{{ error }}</p>
    <p v-else-if="loading && !report" class="text-xs text-slate-500">Loading…</p>
    <p v-else-if="!report || report.usage.length === 0" class="text-xs text-slate-500">
      No model calls recorded in the last 24 hours.
    </p>
    <table v-else class="w-full text-xs">
      <thead>
        <tr class="text-left text-slate-500">
          <th class="font-medium pb-1">Provider</th>
          <th class="font-medium pb-1">Model</th>
          <th class="font-medium pb-1">Role</th>
          <th class="font-medium pb-1 text-right">Calls</th>
          <th class="font-medium pb-1 text-right">Input</th>
          <th class="font-medium pb-1 text-right">Output</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="row in report.usage" :key="row.provider + row.model + row.role">
          <td class="py-0.5 text-slate-200 truncate max-w-[10rem]">{{ row.provider }}</td>
          <td class="py-0.5 text-slate-400 font-mono truncate max-w-[10rem]">
            {{ row.model || '—' }}
          </td>
          <td class="py-0.5 text-slate-400">{{ row.role || '—' }}</td>
          <td class="py-0.5 text-right">{{ row.calls }}</td>
          <td class="py-0.5 text-right">{{ format(row.inputTokens) }}</td>
          <td class="py-0.5 text-right">{{ format(row.outputTokens) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { useUsage } from '../composables/useUsage';

const { report, loading, error } = useUsage('24h');

/** Formats a token count: 1 234 -> 1.2k, 1 500 000 -> 1.5M. */
function format(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`;
  return String(n);
}
</script>
