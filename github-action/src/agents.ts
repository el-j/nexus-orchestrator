import * as https from 'https';
import * as core from '@actions/core';

const AGENCY_BASE = 'https://raw.githubusercontent.com/el-j/agency-agents';

export const AGENT_CATEGORIES = [
  'design',
  'engineering',
  'game-development',
  'marketing',
  'paid-media',
  'product',
  'project-management',
  'testing',
  'support',
  'spatial-computing',
  'specialized',
] as const;

export type AgentCategory = (typeof AGENT_CATEGORIES)[number];

export interface AgentIdentity {
  readonly name: string;
  readonly slug: string;
  readonly description: string;
  readonly color: string;
  readonly category: string;
  readonly systemPrompt: string;
}

/** Convert a display name or slug to a filesystem slug */
export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '');
}

/** Parse YAML-like frontmatter (key: value lines only, no nested structures) */
export function parseFrontmatter(raw: string): {
  data: Record<string, string>;
  body: string;
} {
  const lines = raw.split('\n');
  if (lines[0]?.trimEnd() !== '---') {
    return { data: {}, body: raw };
  }
  let closingIdx = -1;
  for (let i = 1; i < lines.length; i++) {
    if (lines[i]?.trimEnd() === '---') {
      closingIdx = i;
      break;
    }
  }
  if (closingIdx === -1) {
    return { data: {}, body: raw };
  }
  const data: Record<string, string> = {};
  for (const line of lines.slice(1, closingIdx)) {
    const idx = line.indexOf(': ');
    if (idx !== -1) {
      data[line.slice(0, idx).trim()] = line.slice(idx + 2).trim();
    }
  }
  return { data, body: lines.slice(closingIdx + 1).join('\n') };
}

interface FetchResult {
  statusCode: number;
  contentType: string;
  retryAfter: string | undefined;
  rateLimitRemaining: string | undefined;
  body: string;
}

const MAX_REDIRECTS = 5;
const REDIRECT_CODES = new Set([301, 302, 307, 308]);

/**
 * GET with full status/header access. Follows up to MAX_REDIRECTS redirects
 * (relative Locations included), never forwards request headers to another host
 * (they may carry a token), and destroys the socket on timeout.
 */
function request(
  url: string,
  headers: Record<string, string> = {},
  redirectsLeft = MAX_REDIRECTS,
): Promise<FetchResult> {
  return new Promise((resolve, reject) => {
    const req = https.get(url, { timeout: 10_000, headers }, (res) => {
      const statusCode = res.statusCode ?? 0;
      const loc = res.headers.location;
      if (REDIRECT_CODES.has(statusCode) && loc != null && loc.length > 0) {
        res.resume();
        if (redirectsLeft <= 0) {
          reject(new Error(`Too many redirects for ${url}`));
          return;
        }
        const next = new URL(loc, url);
        const sameHost = next.host === new URL(url).host;
        resolve(request(next.toString(), sameHost ? headers : {}, redirectsLeft - 1));
        return;
      }
      const chunks: Uint8Array[] = [];
      res.on('data', (c: Uint8Array) => chunks.push(c));
      res.on('end', () =>
        resolve({
          statusCode,
          contentType: (res.headers['content-type'] as string | undefined) ?? '',
          retryAfter: res.headers['retry-after'] as string | undefined,
          rateLimitRemaining: res.headers['x-ratelimit-remaining'] as string | undefined,
          body: Buffer.concat(chunks).toString('utf-8'),
        }),
      );
      res.on('error', reject);
    });
    req.on('error', reject);
    // The 'timeout' event only reports inactivity; the socket must be destroyed explicitly.
    req.on('timeout', () => req.destroy(new Error(`Request timed out: ${url}`)));
  });
}

/** HTTP GET — returns response body as string; rejects on any non-200 status */
export async function httpsGet(url: string): Promise<string> {
  const res = await request(url);
  if (res.statusCode !== 200) {
    throw new Error(`HTTP ${res.statusCode === 0 ? '?' : res.statusCode} for ${url}`);
  }
  return res.body;
}

/** Fetch a single agent markdown file and parse it */
export async function fetchAgent(
  category: string,
  filename: string,
  ref = 'main',
): Promise<AgentIdentity | null> {
  const url = `${AGENCY_BASE}/${ref}/${category}/${filename}`;
  let raw: string;
  try {
    raw = await httpsGet(url);
  } catch {
    return null;
  }
  const { data, body } = parseFrontmatter(raw);
  if (!data['name'] || !data['description']) return null;
  const name = data['name'];
  return {
    name,
    slug: slugify(name),
    description: data['description'],
    color: data['color'] ?? 'gray',
    category,
    systemPrompt: body.trim(),
  };
}

/**
 * Fetch the full agent index for a category by listing the GitHub API tree.
 * Logs actionable warnings on rate-limit (403/429) or HTML responses instead
 * of failing silently.
 */
export async function fetchCategoryIndex(category: string, ref = 'main'): Promise<string[]> {
  const url = `https://api.github.com/repos/el-j/agency-agents/contents/${category}?ref=${ref}`;
  // api.github.com rejects requests without a User-Agent (403), and unauthenticated
  // callers share a 60 requests/hour limit, so use the workflow token when present.
  const headers: Record<string, string> = {
    'User-Agent': 'nexus-orchestrator-action',
    Accept: 'application/vnd.github+json',
  };
  const token = process.env['GITHUB_TOKEN'];
  if (token != null && token.length > 0) headers['Authorization'] = `Bearer ${token}`;

  let result: FetchResult;
  try {
    result = await request(url, headers);
  } catch {
    return [];
  }

  const { statusCode, contentType, retryAfter, rateLimitRemaining, body } = result;

  // GitHub returns HTML when the API rate limit is exceeded without an Accept header
  if (contentType.startsWith('text/html')) {
    core.warning('GitHub API rate limit hit — agent discovery skipped');
    return [];
  }

  if (statusCode === 403 && rateLimitRemaining === '0') {
    core.warning('GitHub API rate limit exhausted (403) — set GITHUB_TOKEN to raise the limit');
    return [];
  }

  if (statusCode === 403) {
    core.warning('GitHub API access denied (403) — check GITHUB_TOKEN permissions');
    return [];
  }

  if (statusCode === 429) {
    core.warning(
      'GitHub API rate limited (429) — retry after ' +
        (retryAfter != null ? retryAfter + 's' : 'unknown'),
    );
    return [];
  }

  if (statusCode !== 200) {
    return [];
  }

  try {
    const items = JSON.parse(body) as Array<{ name: string; type: string }>;
    return items.filter((i) => i.type === 'file' && i.name.endsWith('.md')).map((i) => i.name);
  } catch {
    return [];
  }
}

/**
 * Resolve one or more agent slugs/names to AgentIdentity objects.
 *
 * Strategy:
 *   1. Try category prefix: "engineering-backend-architect" → category="engineering", file="engineering-backend-architect.md"
 *   2. Try direct filename match across all categories
 *   3. Slug match against all files in all categories (expensive but reliable)
 */
export async function resolveAgents(
  slugsOrNames: string[],
  ref = 'main',
): Promise<AgentIdentity[]> {
  const results: AgentIdentity[] = [];
  for (const input of slugsOrNames) {
    const norm = slugify(input.trim());
    // Fast path: slug already has a known category prefix
    const matchingCategory = AGENT_CATEGORIES.find((cat) => norm.startsWith(cat + '-'));
    if (matchingCategory != null) {
      const filename = `${norm}.md`;
      const agent = await fetchAgent(matchingCategory, filename, ref);
      if (agent != null) {
        results.push(agent);
        continue;
      }
    }
    // Slow path: scan all categories
    let found = false;
    for (const cat of AGENT_CATEGORIES) {
      const files = await fetchCategoryIndex(cat, ref);
      for (const file of files) {
        if (file === `${norm}.md` || file.replace(/\.md$/, '') === norm) {
          const agent = await fetchAgent(cat, file, ref);
          if (agent != null) {
            results.push(agent);
            found = true;
            break;
          }
        }
      }
      if (found) break;
    }
    if (!found) {
      throw new Error(
        `Agent "${input}" not found in el-j/agency-agents@${ref}. ` +
          `Check slugs at https://github.com/el-j/agency-agents`,
      );
    }
  }
  return results;
}

/** Fetch ALL agents from a given category */
export async function resolveCategory(category: string, ref = 'main'): Promise<AgentIdentity[]> {
  if (!(AGENT_CATEGORIES as readonly string[]).includes(category)) {
    throw new Error(`Unknown category "${category}". Valid: ${AGENT_CATEGORIES.join(', ')}`);
  }
  const files = await fetchCategoryIndex(category, ref);
  const agents = await Promise.all(files.map((f) => fetchAgent(category, f, ref)));
  return agents.filter((a): a is AgentIdentity => a !== null);
}

/** Build a combined swarm orchestrator prompt from multiple agents */
export function buildSwarmPrompt(
  agents: AgentIdentity[],
  swarmName = 'Agency Swarm',
  mission = '',
): string {
  const summaries = agents
    .map((a) => `### ${a.name} (${a.category})\n> ${a.description}`)
    .join('\n\n');
  const names = agents.map((a) => a.name).join(', ');
  return [
    `# ${swarmName}`,
    '',
    mission.length > 0 ? `**Mission**: ${mission}\n` : '',
    `You are coordinating a swarm of ${agents.length} specialist AI agents: ${names}.`,
    '',
    '## Agent Roster',
    '',
    summaries,
    '',
    '## Coordination Rules',
    '',
    '1. Assign tasks to the most qualified agent based on their specialty.',
    '2. Handoff context clearly between agents — include relevant prior outputs.',
    '3. One agent per task: avoid overlapping responsibilities.',
    '4. Quality gates: each agent output must be reviewed before advancing.',
    '5. Escalate blockers immediately rather than stalling the pipeline.',
    '',
    '## Agent System Prompts',
    '',
    ...agents.map((a) => `### ${a.name}\n\n<system-prompt>\n${a.systemPrompt}\n</system-prompt>`),
  ]
    .join('\n')
    .trim();
}
