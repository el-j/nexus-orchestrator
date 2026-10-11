import { describe, it, expect } from 'vitest';
import { renderMarkdown, slugify, stripFrontMatter } from './markdown';

describe('slugify', () => {
  it.each([
    ['Get Brain Status', 'get-brain-status'],
    ['  Padded   spaces  ', 'padded-spaces'],
    ['Auth & local-access (v2)!', 'auth-local-access-v2'],
    ['`POST /api/tasks`', 'post-apitasks'],
    ['Tasks &amp; Drafts', 'tasks-drafts'],
    ['<em>HTML</em> inside', 'html-inside'],
    ['---', ''],
  ])('%s → %s', (input, expected) => expect(slugify(input)).toBe(expected));
});

describe('stripFrontMatter', () => {
  it('removes a leading YAML block, including with CRLF line endings', () => {
    expect(stripFrontMatter('---\ntitle: x\n---\n# Hi')).toBe('# Hi');
    expect(stripFrontMatter('---\r\ntitle: x\r\n---\r\n# Hi')).toBe('# Hi');
  });
  it('leaves a document without front matter, or with a mid-document rule, untouched', () => {
    expect(stripFrontMatter('# Hi\n\n---\n\ntext')).toBe('# Hi\n\n---\n\ntext');
    expect(stripFrontMatter('')).toBe('');
  });
});

describe('renderMarkdown', () => {
  it('renders GitHub-flavoured markdown: tables, fenced code, inline code', () => {
    const { html } = renderMarkdown('| a | b |\n|---|---|\n| `x` | y |\n\n```json\n{"k":1}\n```\n');
    expect(html).toContain('<table>');
    expect(html).toContain('<code>x</code>');
    expect(html).toContain('<pre><code class="language-json">');
  });

  it('gives every heading an id and builds a table of contents from ## and ###', () => {
    const { html, toc } = renderMarkdown('# Title\n\n## Alpha\n\n### Beta `code`\n\n#### Deep\n');
    expect(toc).toEqual([
      { id: 'alpha', label: 'Alpha', depth: 2 },
      { id: 'beta-code', label: 'Beta code', depth: 3 },
    ]);
    expect(html).toContain('<h2 id="alpha">Alpha</h2>');
    expect(html).toContain('<h3 id="beta-code">Beta <code>code</code></h3>');
    expect(html).toContain('<h1 id="title">Title</h1>'); // present, but not in the TOC
    expect(html).toContain('<h4 id="deep">Deep</h4>');
  });

  it('keeps ids unique when titles repeat', () => {
    const { toc } = renderMarkdown('## Errors\n\n### Errors\n\n### Errors\n');
    expect(toc.map((t) => t.id)).toEqual(['errors', 'errors-2', 'errors-3']);
  });

  it('falls back to a generic id for a heading with no usable characters', () => {
    const { toc } = renderMarkdown('## ***\n');
    expect(toc[0].id).toBe('section');
  });

  it('ignores front matter and renders an empty document', () => {
    expect(renderMarkdown('---\nlayout: x\n---\n## Real').toc.map((t) => t.label)).toEqual([
      'Real',
    ]);
    expect(renderMarkdown('')).toEqual({ html: '', toc: [] });
  });

  it('does not share state between calls', () => {
    renderMarkdown('## Same');
    expect(renderMarkdown('## Same').toc[0].id).toBe('same');
  });
});
