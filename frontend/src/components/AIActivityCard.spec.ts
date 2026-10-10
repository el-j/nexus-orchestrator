import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import AIActivityCard from './AIActivityCard.vue';
import type { AIActivity } from '../types/domain';

function activity(overrides: Partial<AIActivity> = {}): AIActivity {
  return {
    id: 'a1',
    agentName: 'claude',
    activityType: 'message',
    summary: 'User prompt',
    timestamp: new Date().toISOString(),
    ...overrides,
  } as AIActivity;
}

describe('AIActivityCard', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date('2026-06-15T12:00:00Z'));
  });
  afterEach(() => vi.useRealTimers());

  it.each([
    ['message', '💬', 'border-blue-400', 'text-blue-400'],
    ['tool_use', '🔧', 'border-green-400', 'text-green-400'],
    ['thinking', '🧠', 'border-purple-400', 'text-purple-400'],
    ['file_edit', '📄', 'border-orange-400', 'text-orange-400'],
    ['generation', '⚡', 'border-yellow-400', 'text-yellow-400'],
  ])('styles %s activities with %s', (type, emoji, border, accent) => {
    const w = mount(AIActivityCard, {
      props: { activity: activity({ activityType: type as AIActivity['activityType'] }) },
    });
    expect(w.text()).toContain(emoji);
    expect(w.classes()).toContain(border);
    expect(w.find(`.${accent}`).text()).toBe('claude');
  });

  it('has no decoration for an unknown activity type', () => {
    const w = mount(AIActivityCard, {
      props: { activity: activity({ activityType: 'mystery' as AIActivity['activityType'] }) },
    });
    expect(w.classes().some((c) => c.startsWith('border-') && c !== 'border-l-4')).toBe(false);
  });

  it('shows model, relative time, summary styling per type', () => {
    const w = mount(AIActivityCard, {
      props: {
        activity: activity({
          model: 'opus',
          activityType: 'tool_use',
          timestamp: new Date(Date.now() - 5 * 60_000).toISOString(),
        }),
      },
    });
    expect(w.text()).toContain('· opus');
    expect(w.text()).toContain('5 min ago');
    expect(w.find('.font-mono').exists()).toBe(true);
    const thinking = mount(AIActivityCard, {
      props: { activity: activity({ activityType: 'thinking' }) },
    });
    expect(thinking.find('.italic').exists()).toBe(true);
  });

  it('formats the token total and hides it when zero', () => {
    const w = mount(AIActivityCard, {
      props: { activity: activity({ tokensIn: 1200, tokensOut: 345 }) },
    });
    expect(w.text()).toContain((1545).toLocaleString());
    expect(mount(AIActivityCard, { props: { activity: activity() } }).text()).not.toContain('🪙');
  });

  it('shows only the last two path segments, including for Windows paths', () => {
    expect(
      mount(AIActivityCard, {
        props: { activity: activity({ projectPath: '/Users/me/work/app' }) },
      }).text(),
    ).toContain('work/app');
    expect(
      mount(AIActivityCard, {
        props: { activity: activity({ projectPath: 'C:\\code\\proj' }) },
      }).text(),
    ).toContain('code/proj');
    expect(mount(AIActivityCard, { props: { activity: activity() } }).text()).not.toContain('/');
  });

  it('puts every detail into the hover title', () => {
    const w = mount(AIActivityCard, {
      props: {
        activity: activity({
          projectPath: '/p/x',
          model: 'opus',
          tokensIn: 3,
          tokensOut: 4,
          metadata: { branch: 'main' },
        }),
      },
    });
    const title = w.attributes('title')!;
    for (const part of [
      'Agent: claude',
      'Type: message',
      'Project: /p/x',
      'Model: opus',
      'Tokens in: 3',
      'Tokens out: 4',
      'branch: main',
    ]) {
      expect(title).toContain(part);
    }
  });
});
