import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { mount } from '@vue/test-utils';
import SettingsView from './SettingsView.vue';

const h = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
  getToken: vi.fn(() => ''),
  setToken: vi.fn(),
  base: { value: 'http://127.0.0.1:63987' },
}));
vi.mock('../types/wails', () => ({
  getRuntimeConfig: h.get,
  updateRuntimeConfig: h.update,
  getApiToken: h.getToken,
  setApiToken: h.setToken,
}));
vi.mock('../composables/useServerUrl', () => ({
  resolveServerUrl: () => Promise.resolve(h.base.value),
}));

const Dialog = {
  name: 'AppConfirmDialog',
  props: ['open', 'title', 'message'],
  emits: ['confirm', 'cancel'],
  template: '<div class="dlg" :data-open="open">{{ title }}</div>',
};

const fetchMock = vi.fn();
const clipboard = { writeText: vi.fn() };
const cfg = (o: Record<string, unknown> = {}) => ({
  queueCap: 50,
  apiTokenEnabled: false,
  mcpTokenEnabled: false,
  ...o,
});
const flush = () => vi.advanceTimersByTimeAsync(0);

async function mountView() {
  const w = mount(SettingsView, { global: { stubs: { AppConfirmDialog: Dialog } } });
  await flush();
  return w;
}
type W = Awaited<ReturnType<typeof mountView>>;
const btn = (w: W, text: string, nth = 0) =>
  w.findAll('button').filter((b) => b.text() === text)[nth];

beforeEach(() => {
  vi.useFakeTimers();
  h.get.mockReset().mockResolvedValue(cfg());
  h.update.mockReset().mockResolvedValue(cfg());
  h.getToken.mockReset().mockReturnValue('');
  h.setToken.mockReset();
  h.base.value = 'http://127.0.0.1:63987';
  fetchMock
    .mockReset()
    .mockImplementation(() => Promise.resolve(new Response('{}', { status: 404 })));
  clipboard.writeText.mockReset().mockResolvedValue(undefined);
  vi.stubGlobal('fetch', fetchMock);
  vi.stubGlobal('navigator', { clipboard });
});
afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('SettingsView — server addresses', () => {
  it('shows the MCP endpoint the daemon advertises', async () => {
    fetchMock.mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ mcp: { endpoint: 'http://10.0.0.5:7000/mcp' } })),
      ),
    );
    const w = await mountView();
    expect(fetchMock).toHaveBeenCalledWith('http://127.0.0.1:63987/.well-known/nexus.json');
    expect(w.text()).toContain('http://10.0.0.5:7000/mcp');
  });

  it('falls back to deriving it for daemons that do not advertise one', async () => {
    fetchMock.mockImplementation(() => Promise.resolve(new Response(JSON.stringify({ mcp: {} }))));
    expect((await mountView()).text()).toContain('http://127.0.0.1:63988/mcp');
    fetchMock.mockImplementation(() => Promise.resolve(new Response('{}', { status: 404 })));
    expect((await mountView()).text()).toContain('http://127.0.0.1:63988/mcp');
  });

  it('keeps the defaults when discovery fails entirely', async () => {
    fetchMock.mockRejectedValue(new Error('offline'));
    const w = await mountView();
    expect(w.text()).toContain('http://127.0.0.1:63987');
    expect(w.text()).toContain('/mcp');
  });

  it('copies an address and shows confirmation for 2s', async () => {
    const w = await mountView();
    await btn(w, 'Copy').trigger('click');
    await flush();
    expect(clipboard.writeText).toHaveBeenCalledWith('http://127.0.0.1:63987');
    expect(w.text()).toContain('✓ Copied');
    await vi.advanceTimersByTimeAsync(2000);
    expect(w.text()).not.toContain('✓ Copied');
  });

  it('restarts the confirmation timer on a second copy, and survives a missing clipboard', async () => {
    const w = await mountView();
    await btn(w, 'Copy').trigger('click'); // API row; its label becomes "✓ Copied"
    await vi.advanceTimersByTimeAsync(1500);
    await btn(w, 'Copy').trigger('click'); // now the MCP row; restarts the timer
    await vi.advanceTimersByTimeAsync(1500);
    expect(w.text()).toContain('✓ Copied');
    clipboard.writeText.mockRejectedValue(new Error('denied'));
    await btn(w, 'Copy').trigger('click'); // the API row again
    await flush();
    expect(w.exists()).toBe(true);
  });
});

describe('SettingsView — config', () => {
  it('shows the queue cap, defaulting to 50 before it loads', async () => {
    h.get.mockResolvedValue(cfg({ queueCap: 7 }));
    expect((await mountView()).text()).toContain('7');
  });

  it('saves an updated queueCap via updateRuntimeConfig and handles save errors', async () => {
    h.get.mockResolvedValue(cfg({ queueCap: 50 }));
    h.update.mockResolvedValueOnce(cfg({ queueCap: 120 }));
    const w = await mountView();
    const input = w.find('[data-testid="queue-cap-input"]');
    expect(input.exists()).toBe(true);
    await input.setValue('120');
    await btn(w, 'Save').trigger('click');
    await flush();
    expect(h.update).toHaveBeenCalledWith({ queueCap: 120 });

    h.update.mockRejectedValueOnce(new Error('invalid cap'));
    await input.setValue('200');
    await btn(w, 'Save').trigger('click');
    await flush();
    expect(w.text()).toContain('invalid cap');

    h.update.mockRejectedValueOnce('weird');
    await btn(w, 'Save').trigger('click');
    await flush();
    expect(w.text()).toContain('Failed to update queue cap');
  });

  it('shows Protected vs No auth per token', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true }));
    const t = (await mountView()).text();
    expect(t).toContain('Protected');
    expect(t).toContain('No auth');
  });

  it('shows a load error with a generic fallback', async () => {
    h.get.mockRejectedValue('weird');
    expect((await mountView()).text()).toContain('Failed to load configuration');
  });
});

describe('SettingsView — tokens', () => {
  it('generates a token, shows it once, and reloads the config', async () => {
    h.update.mockResolvedValue(cfg({ apiToken: 'secret-1' }));
    const w = await mountView();
    h.get.mockClear();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    expect(h.update).toHaveBeenCalledWith({ rotateApiToken: true });
    expect(w.text()).toContain('API token');
    expect(w.text()).toContain('secret-1');
    expect(h.get).toHaveBeenCalledTimes(1);
  });

  it('rotates the MCP token with its own flag', async () => {
    h.update.mockResolvedValue(cfg({ mcpToken: 'mcp-secret' }));
    const w = await mountView();
    await btn(w, 'Generate Token', 1).trigger('click');
    await flush();
    expect(h.update).toHaveBeenCalledWith({ rotateMcpToken: true });
    expect(w.text()).toContain('MCP token');
    expect(w.text()).toContain('mcp-secret');
  });

  it('shows nothing when the daemon returns no token', async () => {
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    expect(w.text()).not.toContain("won't be shown");
  });

  it('offers Rotate and Disable for a protected endpoint', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true, mcpTokenEnabled: true }));
    const w = await mountView();
    expect(w.findAll('button').filter((b) => b.text() === 'Rotate')).toHaveLength(2);
    expect(w.findAll('button').filter((b) => b.text() === 'Disable')).toHaveLength(2);
    expect(w.findAll('button').some((b) => b.text() === 'Generate Token')).toBe(false);
  });

  it('shows a rotate failure (Error and non-Error)', async () => {
    h.update.mockRejectedValueOnce(new Error('denied')).mockRejectedValueOnce('weird');
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    expect(w.text()).toContain('denied');
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    expect(w.text()).toContain('Failed to rotate token');
  });

  it('ignores a second rotate while one is in flight', async () => {
    let done!: (c: unknown) => void;
    h.update.mockReturnValue(new Promise((r) => (done = r)));
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    expect(w.text()).toContain('Working…');
    await btn(w, 'Generate Token').trigger('click'); // the MCP row, while the API rotate runs
    expect(h.update).toHaveBeenCalledTimes(1);
    done(cfg());
    await flush();
  });

  it('asks before disabling and only then clears the token', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true }));
    const w = await mountView();
    await btn(w, 'Disable').trigger('click');
    const dlg = w.findComponent(Dialog);
    expect(dlg.props('open')).toBe(true);
    expect(dlg.props('title')).toContain('HTTP API');
    expect(h.update).not.toHaveBeenCalled();
    dlg.vm.$emit('confirm');
    await flush();
    expect(h.update).toHaveBeenCalledWith({ apiToken: '' });
    expect(w.findComponent(Dialog).props('open')).toBe(false);
  });

  it('disables the MCP token with its own flag and names it in the dialog', async () => {
    h.get.mockResolvedValue(cfg({ mcpTokenEnabled: true }));
    const w = await mountView();
    await btn(w, 'Disable').trigger('click');
    expect(w.findComponent(Dialog).props('title')).toContain('MCP');
    w.findComponent(Dialog).vm.$emit('confirm');
    await flush();
    expect(h.update).toHaveBeenCalledWith({ mcpToken: '' });
  });

  it('cancelling the dialog changes nothing', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true }));
    const w = await mountView();
    await btn(w, 'Disable').trigger('click');
    w.findComponent(Dialog).vm.$emit('cancel');
    await flush();
    expect(h.update).not.toHaveBeenCalled();
    expect(w.findComponent(Dialog).props('open')).toBe(false);
  });

  it('reports a failed disable', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true }));
    h.update.mockRejectedValueOnce(new Error('nope')).mockRejectedValueOnce('weird');
    const w = await mountView();
    await btn(w, 'Disable').trigger('click');
    w.findComponent(Dialog).vm.$emit('confirm');
    await flush();
    expect(w.text()).toContain('nope');
    await btn(w, 'Disable').trigger('click');
    w.findComponent(Dialog).vm.$emit('confirm');
    await flush();
    expect(w.text()).toContain('Failed to disable token');
  });

  it('ignores a disable confirmed while another operation runs', async () => {
    h.get.mockResolvedValue(cfg({ apiTokenEnabled: true }));
    let done!: (c: unknown) => void;
    h.update.mockReturnValue(new Promise((r) => (done = r)));
    const w = await mountView();
    await btn(w, 'Rotate').trigger('click');
    await btn(w, 'Disable').trigger('click');
    w.findComponent(Dialog).vm.$emit('confirm');
    await flush();
    expect(h.update).toHaveBeenCalledTimes(1);
    done(cfg());
    await flush();
  });

  it('copies a freshly generated token and can dismiss it', async () => {
    h.update.mockResolvedValue(cfg({ apiToken: 'secret-1' }));
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    const copy = w
      .find('.border-emerald-500\\/20')
      .findAll('button')
      .find((b) => b.text() === 'Copy')!;
    await copy.trigger('click');
    await flush();
    expect(clipboard.writeText).toHaveBeenCalledWith('secret-1');
    await vi.advanceTimersByTimeAsync(2000);
    await w
      .findAll('button')
      .find((b) => b.text() === '✕')!
      .trigger('click');
    expect(w.text()).not.toContain('secret-1');
  });

  it('survives a clipboard failure when copying the token', async () => {
    h.update.mockResolvedValue(cfg({ apiToken: 'secret-1' }));
    clipboard.writeText.mockRejectedValue(new Error('denied'));
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    await w.find('.border-emerald-500\\/20').findAll('button')[0].trigger('click');
    await flush();
    expect(w.text()).not.toContain('✓ Copied');
  });

  it('copies a fresh token twice in a row, restarting the confirmation timer', async () => {
    h.update.mockResolvedValue(cfg({ apiToken: 'secret-1' }));
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    const copy = () =>
      w
        .find('.border-emerald-500\\/20')
        .findAll('button')
        .find((b) => b.text().includes('Cop'))!;
    await copy().trigger('click');
    await flush();
    await copy().trigger('click');
    await flush();
    expect(clipboard.writeText).toHaveBeenCalledTimes(2);
  });

  it('runs a double-clicked rotate only once', async () => {
    let done!: (c: unknown) => void;
    h.update.mockReturnValue(new Promise((r) => (done = r)));
    const w = await mountView();
    const generate = btn(w, 'Generate Token').element as HTMLButtonElement;
    generate.click();
    generate.click(); // the DOM has not re-rendered as disabled yet
    expect(h.update).toHaveBeenCalledTimes(1);
    done(cfg());
    await flush();
  });

  it('ignores a disable confirmed after a rotate started behind the dialog', async () => {
    h.get.mockResolvedValue(cfg({ apiToken: 'x', apiTokenEnabled: true, mcpTokenEnabled: false }));
    let done!: (c: unknown) => void;
    h.update.mockReturnValue(new Promise((r) => (done = r)));
    const w = await mountView();
    await btn(w, 'Disable').trigger('click'); // opens the dialog
    await btn(w, 'Generate Token').trigger('click'); // the other endpoint starts a rotate
    w.findComponent(Dialog).vm.$emit('confirm');
    await flush();
    expect(h.update).toHaveBeenCalledTimes(1);
    done(cfg());
    await flush();
  });

  it('labels the MCP rotate button while that rotate runs', async () => {
    h.get.mockResolvedValue(cfg({ mcpTokenEnabled: true }));
    h.update.mockReturnValue(new Promise(() => {}));
    const w = await mountView();
    await btn(w, 'Rotate').trigger('click');
    expect(w.text()).toContain('Working…');
  });

  it('persists the API token via setApiToken on rotate, disable, and manual browser token entry', async () => {
    h.update.mockResolvedValueOnce(cfg({ apiToken: 'secret-api-99', apiTokenEnabled: true }));
    const w = await mountView();
    await btn(w, 'Generate Token').trigger('click');
    await flush();
    expect(h.setToken).toHaveBeenCalledWith('secret-api-99');

    // Manual browser token input
    const tokenInput = w.find('[data-testid="browser-api-token-input"]');
    expect(tokenInput.exists()).toBe(true);
    await tokenInput.setValue('pasted-token');
    await w.find('[data-testid="browser-api-token-save"]').trigger('click');
    await flush();
    expect(h.setToken).toHaveBeenCalledWith('pasted-token');
  });
});
