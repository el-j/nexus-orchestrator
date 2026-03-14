/**
 * relativeTime — returns a human-readable relative-time string.
 * Unifies relativeTime(), timeAgo(), and formatDate() from multiple components.
 */
export function relativeTime(iso: string | undefined): string {
  if (!iso) return '—';
  const diff = Date.now() - new Date(iso).getTime();
  if (diff < 60_000) return 'just now';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} min ago`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} hr ago`;
  return new Date(iso).toLocaleDateString();
}

/** Alias for call-sites that used `timeAgo`. */
export const timeAgo = relativeTime;

/** Alias for call-sites that used `formatDate`. */
export const formatDate = relativeTime;
