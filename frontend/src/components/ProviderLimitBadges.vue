<!--
  Badges for a provider that is limited: "cooling down until HH:MM" after a rate limit, an exhausted
  quota or a spent daily budget, and the share of the plan used when the provider's client reports it.
  Props: provider.
-->
<template>
  <span class="inline-flex items-center gap-1.5">
    <span
      v-if="provider.rateLimited"
      class="text-[10px] px-1.5 py-0.5 rounded-full bg-amber-500/20 text-amber-300 font-semibold"
      :title="provider.cooldownReason"
      data-testid="cooling-badge"
    >
      cooling down{{ until ? ` until ${until}` : '' }}
    </span>
    <span
      v-if="share !== null"
      class="text-[10px] px-1.5 py-0.5 rounded-full font-semibold"
      :class="share >= 90 ? 'bg-red-500/20 text-red-300' : 'bg-slate-500/20 text-slate-300'"
      :title="`${provider.quota?.window} window`"
      data-testid="quota-badge"
    >
      plan {{ share }}% used
    </span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { ProviderInfo } from '../types/domain';

const props = defineProps<{ provider: ProviderInfo }>();

/** The time the provider is tried again, as the local clock shows it, or '' when unknown. */
const until = computed(() => {
  const iso = props.provider.rateLimitedUntil;
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime())
    ? ''
    : d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
});

/** The share of the plan used in percent, or null when the client does not report one. */
const share = computed<number | null>(() => {
  const u = props.provider.quota?.utilization;
  return u === undefined || u < 0 ? null : Math.round(u * 100);
});
</script>
