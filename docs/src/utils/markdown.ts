import { Marked, type Tokens } from 'marked';

/** One entry of a page's table of contents. */
export interface TocItem {
  id: string;
  label: string;
  /** 2 for `##`, 3 for `###`. */
  depth: 2 | 3;
}

/** Result of rendering a markdown page: the HTML and its table of contents. */
export interface RenderedMarkdown {
  html: string;
  toc: TocItem[];
}

/** Lower-cases a heading into a URL fragment: "Get Brain Status" → "get-brain-status". */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/<[^>]+>/g, '')
    .replace(/&[a-z]+;/g, '')
    .replace(/[^a-z0-9\s-]/g, '')
    .trim()
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '');
}

/** Removes a leading YAML front-matter block (`---` … `---`), if present. */
export function stripFrontMatter(src: string): string {
  const m = /^---\r?\n[\s\S]*?\r?\n---\r?\n?/.exec(src);
  return m ? src.slice(m[0].length) : src;
}

/**
 * Renders a trusted, repository-owned Markdown document (GitHub-flavoured) to
 * HTML and collects a table of contents from its `##` and `###` headings.
 * Heading ids are unique within the document even when titles repeat.
 *
 * The input is build-time content from this repository, never user input, so the
 * output is not sanitised.
 */
export function renderMarkdown(src: string): RenderedMarkdown {
  const toc: TocItem[] = [];
  const seen = new Map<string, number>();

  const marked = new Marked({ gfm: true });
  marked.use({
    renderer: {
      heading(
        this: { parser: { parseInline(t: Tokens.Generic[]): string } },
        token: Tokens.Heading,
      ) {
        const text = token.text;
        let id = slugify(text) || 'section';
        const n = seen.get(id) ?? 0;
        seen.set(id, n + 1);
        if (n > 0) id = `${id}-${n + 1}`;
        if (token.depth === 2 || token.depth === 3) {
          toc.push({ id, label: text.replace(/`/g, ''), depth: token.depth });
        }
        const inner = this.parser.parseInline(token.tokens);
        return `<h${token.depth} id="${id}">${inner}</h${token.depth}>\n`;
      },
    },
  });

  const html = marked.parse(stripFrontMatter(src), { async: false }) as string;
  return { html, toc };
}
