export declare const AGENT_CATEGORIES: readonly ["design", "engineering", "game-development", "marketing", "paid-media", "product", "project-management", "testing", "support", "spatial-computing", "specialized"];
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
export declare function slugify(name: string): string;
/** Parse YAML-like frontmatter (key: value lines only, no nested structures) */
export declare function parseFrontmatter(raw: string): {
    data: Record<string, string>;
    body: string;
};
/** HTTP GET — returns response body as string */
export declare function httpsGet(url: string): Promise<string>;
/** Fetch a single agent markdown file and parse it */
export declare function fetchAgent(category: string, filename: string, ref?: string): Promise<AgentIdentity | null>;
/**
 * Fetch the full agent index for a category by listing the GitHub API tree.
 * Falls back gracefully if the API is rate-limited.
 */
export declare function fetchCategoryIndex(category: string, ref?: string): Promise<string[]>;
/**
 * Resolve one or more agent slugs/names to AgentIdentity objects.
 *
 * Strategy:
 *   1. Try category prefix: "engineering-backend-architect" → category="engineering", file="engineering-backend-architect.md"
 *   2. Try direct filename match across all categories
 *   3. Slug match against all files in all categories (expensive but reliable)
 */
export declare function resolveAgents(slugsOrNames: string[], ref?: string): Promise<AgentIdentity[]>;
/** Fetch ALL agents from a given category */
export declare function resolveCategory(category: string, ref?: string): Promise<AgentIdentity[]>;
/** Build a combined swarm orchestrator prompt from multiple agents */
export declare function buildSwarmPrompt(agents: AgentIdentity[], swarmName?: string, mission?: string): string;
