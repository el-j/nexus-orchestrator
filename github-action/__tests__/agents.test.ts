import {
  slugify,
  parseFrontmatter,
  buildSwarmPrompt,
  AGENT_CATEGORIES,
} from '../src/agents';
import type { AgentIdentity } from '../src/agents';

describe('slugify', () => {
  it('lowercases and replaces spaces', () => {
    expect(slugify('Backend Architect')).toBe('backend-architect');
  });
  it('handles already-slugified input', () => {
    expect(slugify('engineering-frontend-developer')).toBe(
      'engineering-frontend-developer'
    );
  });
  it('strips leading/trailing hyphens', () => {
    expect(slugify('  hello world  ')).toBe('hello-world');
  });
  it('collapses multiple separators', () => {
    expect(slugify('foo--bar___baz')).toBe('foo-bar-baz');
  });
});

describe('parseFrontmatter', () => {
  it('parses a valid frontmatter block', () => {
    const raw = `---\nname: Test Agent\ndescription: Does things\ncolor: blue\n---\n# Body here\n\nParagraph.`;
    const { data, body } = parseFrontmatter(raw);
    expect(data['name']).toBe('Test Agent');
    expect(data['description']).toBe('Does things');
    expect(data['color']).toBe('blue');
    expect(body.trim()).toBe('# Body here\n\nParagraph.');
  });

  it('returns empty data when no frontmatter delimiter', () => {
    const raw = 'No frontmatter here.';
    const { data, body } = parseFrontmatter(raw);
    expect(Object.keys(data).length).toBe(0);
    expect(body).toBe(raw);
  });

  it('returns empty data when closing delimiter is missing', () => {
    const raw = '---\nname: Orphan\n';
    const { data } = parseFrontmatter(raw);
    expect(Object.keys(data).length).toBe(0);
  });

  it('ignores lines without ": " separator', () => {
    const raw = '---\nname: Valid\nnodash\n---\nbody';
    const { data } = parseFrontmatter(raw);
    expect(Object.keys(data).length).toBe(1);
    expect(data['name']).toBe('Valid');
  });
});

describe('AGENT_CATEGORIES', () => {
  it('contains expected categories', () => {
    expect(AGENT_CATEGORIES).toContain('engineering');
    expect(AGENT_CATEGORIES).toContain('design');
    expect(AGENT_CATEGORIES).toContain('testing');
  });
  it('has no duplicates', () => {
    expect(new Set(AGENT_CATEGORIES).size).toBe(AGENT_CATEGORIES.length);
  });
});

describe('buildSwarmPrompt', () => {
  const agents: AgentIdentity[] = [
    {
      name: 'Backend Architect',
      slug: 'backend-architect',
      description: 'Designs backend systems',
      color: 'blue',
      category: 'engineering',
      systemPrompt: 'You are a backend architect.',
    },
    {
      name: 'Frontend Developer',
      slug: 'frontend-developer',
      description: 'Builds UIs',
      color: 'green',
      category: 'engineering',
      systemPrompt: 'You are a frontend developer.',
    },
  ];

  it('includes the swarm name', () => {
    const prompt = buildSwarmPrompt(agents, 'My Swarm');
    expect(prompt).toContain('# My Swarm');
  });

  it('includes all agent names', () => {
    const prompt = buildSwarmPrompt(agents);
    expect(prompt).toContain('Backend Architect');
    expect(prompt).toContain('Frontend Developer');
  });

  it('includes mission when provided', () => {
    const prompt = buildSwarmPrompt(agents, 'Swarm', 'Build a payment system');
    expect(prompt).toContain('Build a payment system');
  });

  it('includes agent system prompts wrapped in tags', () => {
    const prompt = buildSwarmPrompt(agents);
    expect(prompt).toContain('<system-prompt>');
    expect(prompt).toContain('You are a backend architect.');
  });

  it('includes coordination rules', () => {
    const prompt = buildSwarmPrompt(agents);
    expect(prompt).toContain('Coordination Rules');
  });

  it('omits mission section when mission is empty', () => {
    const prompt = buildSwarmPrompt(agents, 'Swarm', '');
    expect(prompt).not.toContain('**Mission**');
  });
});
