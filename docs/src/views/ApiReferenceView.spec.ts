import { describe, it, expect, vi, beforeEach } from 'vitest';
import { mount } from '@vue/test-utils';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import ApiReferenceView from './ApiReferenceView.vue';

const scrollToId = vi.hoisted(() => vi.fn());
vi.mock('../utils/scroll', () => ({ scrollToId }));

const source = readFileSync(resolve(__dirname, '../../api-reference.md'), 'utf8');

beforeEach(() => scrollToId.mockReset());

describe('ApiReferenceView', () => {
  it('renders docs/api-reference.md itself, so the site cannot drift from it', () => {
    const w = mount(ApiReferenceView);
    const doc = w.get('[data-testid="api-doc"]');
    // Every second-level heading of the source document appears on the page.
    const headings = [...source.matchAll(/^## (.+)$/gm)].map((m) => m[1].replace(/`/g, ''));
    expect(headings.length).toBeGreaterThan(3);
    for (const h of headings) expect(doc.text(), h).toContain(h);
  });

  it('shows endpoints that were once missing from the hand-written page', () => {
    const text = mount(ApiReferenceView).get('[data-testid="api-doc"]').text();
    for (const needle of [
      '/api/brain/ingest-content',
      'provider_discovered',
      'NEXUS_INGEST_ALLOWED_DIRS',
      'get_onboarding_context',
    ]) {
      expect(text, needle).toContain(needle);
    }
  });

  it('renders tables and code blocks', () => {
    const doc = mount(ApiReferenceView).get('[data-testid="api-doc"]');
    expect(doc.findAll('table').length).toBeGreaterThan(3);
    expect(doc.findAll('pre code').length).toBeGreaterThan(5);
  });

  it('does not leave Jekyll syntax or front matter visible', () => {
    const text = mount(ApiReferenceView).get('[data-testid="api-doc"]').text();
    expect(text).not.toContain('{:toc}');
    expect(text).not.toContain('{: .no_toc');
    expect(text).not.toContain('layout: default');
  });

  it('builds a sidebar from the headings and scrolls without touching the router hash', async () => {
    const w = mount(ApiReferenceView);
    const links = w.findAll('nav a');
    expect(links.length).toBeGreaterThan(10);
    const first = links[0];
    const id = first.attributes('href')!.slice(1);
    expect(w.get('[data-testid="api-doc"]').find(`[id="${id}"]`).exists()).toBe(true);
    await first.trigger('click');
    expect(scrollToId).toHaveBeenCalledWith(id);
  });

  it('styles top-level and nested entries differently', () => {
    const w = mount(ApiReferenceView);
    const classes = new Set(w.findAll('nav a').map((a) => a.classes().includes('font-semibold')));
    expect(classes).toEqual(new Set([true, false]));
  });

  it('links to its source file', () => {
    const w = mount(ApiReferenceView);
    expect(w.find('a[href$="docs/api-reference.md"]').exists()).toBe(true);
  });
});
