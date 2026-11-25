import { describe, it, expect } from 'vitest';
import { mount } from '@vue/test-utils';
import { defineComponent, ref } from 'vue';
import LadderEditor, { type LadderRoleRow } from './LadderEditor.vue';

const Host = defineComponent({
  components: { LadderEditor },
  props: {
    initial: { type: Array as () => LadderRoleRow[], required: true },
    suggestions: { type: Array as () => string[], default: () => [] },
  },
  setup(props) {
    const rows = ref<LadderRoleRow[]>(props.initial);
    return { rows };
  },
  template: `<LadderEditor v-model="rows" :suggestions="suggestions" />`,
});

describe('LadderEditor', () => {
  it('adds roles, picks suggestions, adds custom rungs, reorders up/down, and removes rungs/roles', async () => {
    const w = mount(Host, {
      props: {
        initial: [{ role: 'coder', rungs: ['Ollama/*', 'Anthropic/claude-sonnet*'] }],
        suggestions: ['Claude (subscription)/haiku', 'Ollama/*'],
      },
    });

    // Reorder second rung up -> swaps order
    const upBtns = w.findAll('[data-testid="ladder-move-up"]');
    await upBtns[1].trigger('click');
    expect(w.vm.rows[0].rungs).toEqual(['Anthropic/claude-sonnet*', 'Ollama/*']);

    // Reorder first rung down -> swaps back
    const downBtns = w.findAll('[data-testid="ladder-move-down"]');
    await downBtns[0].trigger('click');
    expect(w.vm.rows[0].rungs).toEqual(['Ollama/*', 'Anthropic/claude-sonnet*']);

    // Pick a suggestion from the dropdown (and ignore empty selection)
    const select = w.find('[data-testid="ladder-suggestion-select"]');
    await select.setValue('');
    await select.setValue('Claude (subscription)/haiku');
    expect(w.vm.rows[0].rungs).toContain('Claude (subscription)/haiku');

    // Picking a duplicate suggestion does not duplicate it
    await select.setValue('Ollama/*');
    expect(w.vm.rows[0].rungs.filter((r) => r === 'Ollama/*')).toHaveLength(1);

    // Custom draft input via Enter key (comma-separated + blank ignored)
    const input = w.find('[data-testid="ladder-rung-input"]');
    await input.setValue('gpt-5.3-codex, , ');
    await input.trigger('keydown.enter');
    expect(w.vm.rows[0].rungs).toContain('gpt-5.3-codex');

    // Empty draft submit is a no-op
    await w.find('[data-testid="ladder-add-rung"]').trigger('click');

    // Remove a rung
    await w.findAll('[data-testid="ladder-remove-rung"]')[0].trigger('click');
    expect(w.vm.rows[0].rungs).not.toContain('Ollama/*');

    // Add a new role, rename it, then remove it
    await w.find('[data-testid="ladder-add-role"]').trigger('click');
    expect(w.vm.rows).toHaveLength(2);
    const roleInputs = w.findAll('[data-testid="ladder-role-input"]');
    await roleInputs[1].setValue('architect');
    expect(w.vm.rows[1].role).toBe('architect');

    // Trigger empty draft on brand new row (drafts[1] undefined)
    await w.findAll('[data-testid="ladder-add-rung"]')[1].trigger('click');
    await w.findAll('[data-testid="ladder-remove-role"]')[1].trigger('click');
    expect(w.vm.rows).toHaveLength(1);
    w.unmount();
  });
});
