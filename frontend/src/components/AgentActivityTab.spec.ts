import { describe, it, expect, vi } from 'vitest';
import { mount } from '@vue/test-utils';
import { ref } from 'vue';
import AgentActivityTab from './AgentActivityTab.vue';
import type { AIActivity } from '../types/domain';

const activities = ref<AIActivity[]>([]);
const loading = ref(false);

vi.mock('../composables/useActivities', () => ({
  useActivities: () => ({ activities, loading }),
}));

const act = (over: Partial<AIActivity>): AIActivity =>
  ({
    id: 'a',
    agentName: 'Claude Code',
    activityType: 'message',
    summary: 'did a thing',
    timestamp: '2026-06-15T12:00:00Z',
    projectPath: '/work/proj/app',
    ...over,
  }) as AIActivity;

describe('AgentActivityTab', () => {
  it('shows a spinner while loading with no data', () => {
    activities.value = [];
    loading.value = true;
    const w = mount(AgentActivityTab);
    expect(w.find('.animate-spin').exists()).toBe(true);
    loading.value = false;
  });

  it('shows the empty state when nothing was detected', () => {
    activities.value = [];
    const w = mount(AgentActivityTab);
    expect(w.text()).toContain('No AI activity detected');
  });

  it('renders the timeline and filters by agent and project', async () => {
    activities.value = [
      act({ id: '1', agentName: 'Claude Code', summary: 'one' }),
      act({
        id: '2',
        agentName: 'GitHub Copilot',
        activityType: 'tool_use',
        summary: 'two',
        projectPath: '/x/y/z',
      }),
      act({
        id: '3',
        agentName: 'Cursor',
        activityType: 'thinking',
        summary: 'three',
        projectPath: undefined,
      }),
      act({ id: '4', agentName: 'Continue', activityType: 'file_edit', summary: 'four' }),
      act({ id: '5', agentName: 'other', activityType: 'generation', summary: 'five' }),
      act({ id: '6', agentName: 'x', activityType: 'unknown' as never, summary: 'six' }),
    ];
    const w = mount(AgentActivityTab);
    expect(w.text()).toContain('one');
    expect(w.text()).toContain('six');
    const [agent, project] = w.findAll('select');
    await agent.setValue('GitHub Copilot');
    expect(w.text()).toContain('two');
    expect(w.text()).not.toContain('one');
    await agent.setValue('');
    await project.setValue('/x/y/z');
    expect(w.text()).toContain('y/z');
    expect(w.text()).not.toContain('three');
  });
});
