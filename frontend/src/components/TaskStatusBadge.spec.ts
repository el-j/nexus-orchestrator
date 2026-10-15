import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import TaskStatusBadge from './TaskStatusBadge.vue';
import type { TaskStatus } from '../types/domain';

const COLOURS: Record<string, string> = {
  COMPLETED: 'emerald',
  PROCESSING: 'blue',
  QUEUED: 'violet',
  FAILED: 'red',
  CANCELLED: 'slate',
  TOO_LARGE: 'orange',
  NO_PROVIDER: 'yellow',
};

describe('TaskStatusBadge', () => {
  it.each(Object.entries(COLOURS))('renders %s with its %s palette', (status, colour) => {
    const wrapper = mount(TaskStatusBadge, { props: { status: status as TaskStatus } });
    expect(wrapper.text()).toContain(status);
    expect(wrapper.classes().join(' ')).toContain(colour);
  });

  it('falls back to the neutral palette for statuses it does not know', () => {
    const wrapper = mount(TaskStatusBadge, { props: { status: 'DRAFT' as TaskStatus } });
    expect(wrapper.text()).toBe('DRAFT');
    expect(wrapper.classes().join(' ')).toContain('slate');
  });

  it('shows the pulsing dot only while PROCESSING', () => {
    expect(
      mount(TaskStatusBadge, { props: { status: 'PROCESSING' } })
        .find('.animate-pulse')
        .exists(),
    ).toBe(true);
    expect(
      mount(TaskStatusBadge, { props: { status: 'QUEUED' } })
        .find('.animate-pulse')
        .exists(),
    ).toBe(false);
  });
});
