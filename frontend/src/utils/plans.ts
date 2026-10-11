import type { DiscoveredPlanFile, PlanFileKind } from '../types/domain';

/** Display label for a plan-file kind. */
export function kindLabel(kind: PlanFileKind): string {
  const labels: Record<PlanFileKind, string> = {
    nexus: 'Nexus',
    'claude-task': 'Tasks',
    claude: 'Instructions',
    cursor: 'Cursor',
    'mcp-config': 'Config',
    crewai: 'CrewAI',
    markdown: 'Markdown',
  };
  return labels[kind] ?? kind;
}

/** The `PLAN-n` id named in a task file's summary, or null. */
export function extractParentPlan(file: DiscoveredPlanFile): string | null {
  const m = file.summary?.match(/\*\*Plan:\*\*\s*(PLAN-\d+)/);
  if (m) return m[1];
  return null;
}

/** Last path segment. */
export function basename(path: string): string {
  return path.slice(path.lastIndexOf('/') + 1);
}

/** Shortens text to `max` characters, adding an ellipsis. */
export function truncate(text: string, max: number): string {
  return text.length > max ? text.slice(0, max) + '…' : text;
}

/** Locale date for an ISO timestamp; returns the input if it cannot be formatted. */
export function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return iso;
  }
}

/** Badge colour classes per plan-file kind. */
export const kindColors: Record<PlanFileKind, string> = {
  nexus: 'bg-indigo-500/20 text-indigo-300',
  markdown: 'bg-emerald-500/20 text-emerald-300',
  cursor: 'bg-blue-500/20 text-blue-300',
  claude: 'bg-orange-500/20 text-orange-300',
  'mcp-config': 'bg-purple-500/20 text-purple-300',
  crewai: 'bg-pink-500/20 text-pink-300',
  'claude-task': 'bg-amber-500/20 text-amber-300',
};

/** Badge classes for a kind, with a neutral fallback. */
export function kindBadgeClass(kind: PlanFileKind): string {
  return kindColors[kind] ?? 'bg-white/5 text-slate-400';
}

/** Lower-cased status from a plan summary (body `**Status:**` first, then YAML `status:`), or null. */
export function parseStatus(plan: DiscoveredPlanFile): string | null {
  if (!plan.summary) return null;
  // Body-level "**Status:** todo" is checked first: the YAML pattern would otherwise
  // match inside it and capture "** todo".
  const bodyMatch = plan.summary.match(/\*\*Status:\*\*\s*([^\n\r]+)/i);
  if (bodyMatch) return bodyMatch[1].trim().toLowerCase();
  // YAML frontmatter: "status: todo" at the start of a line
  const yamlMatch = plan.summary.match(/^\s*status:\s*([^\n\r]+)/im);
  if (yamlMatch) return yamlMatch[1].trim().toLowerCase();
  return null;
}

/** Badge colour classes per plan status. */
export const statusBadgeColors: Record<string, string> = {
  todo: 'bg-yellow-500/20 text-yellow-300',
  'in-progress': 'bg-blue-500/20 text-blue-300',
  done: 'bg-emerald-500/20 text-emerald-300',
  completed: 'bg-emerald-500/20 text-emerald-300',
  blocked: 'bg-red-500/20 text-red-300',
};

/** Badge classes for a status, with a neutral fallback. */
export function statusBadgeClass(status: string): string {
  return statusBadgeColors[status] ?? 'bg-white/5 text-slate-400';
}
