import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import CodeBlock from './CodeBlock.vue';

beforeEach(() => vi.useFakeTimers());
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('CodeBlock', () => {
  it('shows the code with its language class', () => {
    const w = mount(CodeBlock, { props: { code: 'echo hi', language: 'bash' } });
    expect(w.get('code').text()).toBe('echo hi');
    expect(w.get('code').classes()).toContain('language-bash');
  });

  it('copies to the clipboard and shows a confirmation for two seconds', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal('navigator', { clipboard: { writeText } });
    const w = mount(CodeBlock, { props: { code: 'npm i' } });
    await w.get('button').trigger('click');
    await vi.advanceTimersByTimeAsync(0);
    expect(writeText).toHaveBeenCalledWith('npm i');
    expect(w.get('button').text()).toBe('✓ copied');
    await vi.advanceTimersByTimeAsync(2000);
    expect(w.get('button').text()).toBe('copy');
  });

  it('falls back to a temporary textarea when the clipboard API is unavailable', async () => {
    vi.stubGlobal('navigator', {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error('denied')) },
    });
    const exec = vi.fn().mockReturnValue(true);
    (document as unknown as { execCommand: unknown }).execCommand = exec;
    const w = mount(CodeBlock, { props: { code: 'fallback text' } });
    await w.get('button').trigger('click');
    await vi.advanceTimersByTimeAsync(0);
    expect(exec).toHaveBeenCalledWith('copy');
    expect(document.querySelector('textarea')).toBeNull(); // cleaned up again
    expect(w.get('button').text()).toBe('✓ copied');
  });
});
