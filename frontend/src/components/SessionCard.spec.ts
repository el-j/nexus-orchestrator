import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import SessionCard from './SessionCard.vue';
import type { AISession } from '../types/domain';

const session = (o: Partial<AISession> = {}): AISession =>
  ({
    id: 's1',
    agentName: 'Copilot',
    status: 'active',
    source: 'vscode',
    lastActivity: new Date().toISOString(),
    ...o,
  }) as AISession;

describe('SessionCard', () => {
  it('opens on a click of the card, and the buttons emit their own event without opening it', async () => {
    const w = mount(SessionCard, { props: { session: session() } });
    await w.trigger('click');
    expect(w.emitted('open')).toHaveLength(1);
    for (const [label, event] of [
      ['Delegate', 'delegate'],
      ['Disconnect', 'disconnect'],
      ['Terminate', 'terminate'],
    ] as const) {
      await w
        .findAll('button')
        .find((b) => b.text() === label)!
        .trigger('click');
      expect(w.emitted(event)).toHaveLength(1);
    }
    expect(w.emitted('open')).toHaveLength(1);
  });

  it('shows Delegated instead of the Delegate button, and no actions once disconnected', () => {
    const delegated = mount(SessionCard, {
      props: { session: session({ delegatedToNexus: true }) },
    });
    expect(delegated.text()).toContain('Delegated');
    expect(delegated.text()).not.toContain('Delegate\n');
    const gone = mount(SessionCard, { props: { session: session({ status: 'disconnected' }) } });
    expect(gone.findAll('button')).toHaveLength(0);
  });

  it('shows the source, external id and project', () => {
    const w = mount(SessionCard, {
      props: { session: session({ source: 'mcp', externalId: 'ext-1', projectPath: '/work/app' }) },
    });
    expect(w.text()).toContain('MCP');
    expect(w.text()).toContain('ext-1');
    expect(w.text()).toContain('work/app');
    expect(
      mount(SessionCard, { props: { session: session({ source: 'http' }) } }).text(),
    ).toContain('HTTP');
  });
});
