import { describe, it, expect, vi, beforeEach } from 'vitest';
import { scrollToId } from './scroll';

beforeEach(() => {
  document.body.innerHTML = '';
  vi.restoreAllMocks();
});

describe('scrollToId', () => {
  it('smooth-scrolls to the element with that id', () => {
    document.body.innerHTML = '<h2 id="target">x</h2>';
    const el = document.getElementById('target')!;
    el.scrollIntoView = vi.fn();
    scrollToId('target');
    expect(el.scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth' });
  });

  it('warns instead of throwing when the id does not exist', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});
    expect(() => scrollToId('missing')).not.toThrow();
    expect(warn).toHaveBeenCalledWith('[scrollToId] element #missing not found');
  });
});
