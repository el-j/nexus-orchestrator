import { describe, expect, it } from 'vitest';
import { guideLink, guides } from './guides';

describe('guides', () => {
  it('has unique slugs and real content', () => {
    expect(new Set(guides.map((g) => g.slug)).size).toBe(guides.length);
    for (const g of guides) {
      expect(g.title, g.slug).toBeTruthy();
      expect(g.summary, g.slug).toBeTruthy();
      expect(g.source.length, g.slug).toBeGreaterThan(500);
    }
  });
});

describe('guideLink', () => {
  const base = '/nexus-orchestrator/';
  it('keeps absolute, scheme and in-page links', () => {
    for (const href of ['https://x.dev/a', 'mailto:a@b.c', '#section', '/abs/path']) {
      expect(guideLink(href, base)).toBe(href);
    }
  });

  it('turns a link to another guide into a page of the site, with its anchor', () => {
    expect(guideLink('plan-files.md', base)).toBe('/nexus-orchestrator/guides/plan-files/');
    expect(guideLink('streaming.md#events', base)).toBe(
      '/nexus-orchestrator/guides/streaming/#events',
    );
  });

  it('points everything else at the file in the repository', () => {
    const blob = 'https://github.com/el-j/nexus-orchestrator/blob/main';
    expect(guideLink('api-reference.md#tools', base)).toBe(`${blob}/docs/api-reference.md#tools`);
    expect(guideLink('../vscode-extension/README.md', base)).toBe(
      `${blob}/vscode-extension/README.md`,
    );
    expect(guideLink('../../x/y.md', base)).toBe(`${blob}/x/y.md`);
    expect(guideLink('images/a.png', base)).toBe(`${blob}/docs/images/a.png`);
  });
});
