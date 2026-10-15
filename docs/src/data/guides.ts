/**
 * guides.ts: the guides of the site. Each is a markdown file of the repository's docs/ folder,
 * rendered at build time, so the site and the repository cannot drift apart.
 */
import usingNexus from '../../using-nexus.md?raw';
import howWork from '../../how-work-reaches-nexus.md?raw';
import subscriptions from '../../delegating-subscriptions.md?raw';
import streaming from '../../streaming.md?raw';
import policy from '../../rulebased-workflow.md?raw';
import planFiles from '../../plan-files.md?raw';
import modelBridge from '../../vscode-model-bridge.md?raw';

/** One guide: its page address, its title and summary, and its markdown source. */
export interface Guide {
  slug: string;
  title: string;
  summary: string;
  /** The markdown text. */
  source: string;
}

/** The guides, in the order the index lists them. */
export const guides: Guide[] = [
  {
    slug: 'using-nexus',
    title: 'Using Nexus',
    summary:
      'Run one orchestrator for all your projects: start it, connect your agents, introduce a project.',
    source: usingNexus,
  },
  {
    slug: 'how-work-reaches-nexus',
    title: 'How work reaches Nexus',
    summary: 'What Nexus really does, the one way in for each thing, and what is not a way in.',
    source: howWork,
  },
  {
    slug: 'delegating-subscriptions',
    title: 'Delegating to your subscriptions',
    summary:
      'Claude Pro/Max through the claude CLI, Copilot and other editor models through the VS Code bridge.',
    source: subscriptions,
  },
  {
    slug: 'streaming',
    title: 'Live output',
    summary:
      'See the reply while the model writes it: events, where it shows up, what happens on failure.',
    source: streaming,
  },
  {
    slug: 'rulebased-workflow',
    title: 'Rule-based workflow',
    summary:
      'The project policy (.nexus/policy.json): verification, file limits, tests first, rules, task memory.',
    source: policy,
  },
  {
    slug: 'plan-files',
    title: 'Plans and tasks',
    summary:
      'Where plan and task files may live, how they are found and linked, and how to configure it.',
    source: planFiles,
  },
  {
    slug: 'vscode-model-bridge',
    title: 'VS Code model bridge',
    summary: 'Share the models your editor offers (Copilot and others) with the orchestrator.',
    source: modelBridge,
  },
];

const REPO_BLOB = 'https://github.com/el-j/nexus-orchestrator/blob/main';

/**
 * Where a link of a guide should point on the site. Links between guides become site pages,
 * other relative links point at the file in the repository, absolute and in-page links stay.
 * `base` is the site's base path ("/nexus-orchestrator/").
 */
export function guideLink(href: string, base: string): string {
  if (/^([a-z][a-z0-9+.-]*:|#|\/)/i.test(href)) return href;
  const doc = /^([\w-]+)\.md(#.*)?$/.exec(href);
  if (doc) {
    const hash = doc[2] ?? '';
    return guides.some((g) => g.slug === doc[1])
      ? `${base}guides/${doc[1]}/${hash}`
      : `${REPO_BLOB}/docs/${doc[1]}.md${hash}`;
  }
  if (href.startsWith('../')) return `${REPO_BLOB}/${href.replace(/^(\.\.\/)+/, '')}`;
  return `${REPO_BLOB}/docs/${href}`;
}
