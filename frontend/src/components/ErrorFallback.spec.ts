import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import ErrorFallback from './ErrorFallback.vue';

describe('ErrorFallback', () => {
  it('shows the error message when there is one', () => {
    const wrapper = mount(ErrorFallback, { props: { error: new Error('kaboom') } });
    expect(wrapper.text()).toContain('Something went wrong');
    expect(wrapper.text()).toContain('kaboom');
  });

  it.each([undefined, null])('omits the message block for %s', (error) => {
    const wrapper = mount(ErrorFallback, { props: { error } });
    expect(wrapper.find('.font-mono').exists()).toBe(false);
  });

  it('emits retry when the button is clicked', async () => {
    const wrapper = mount(ErrorFallback);
    await wrapper.find('button').trigger('click');
    expect(wrapper.emitted('retry')).toHaveLength(1);
  });
});
