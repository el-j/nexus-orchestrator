/**
 * A small in-memory stand-in for the `vscode` module, shared by tests that exercise
 * tree providers, views and the activation entry point. Behaviour is deliberately
 * minimal: just what the extension code touches.
 */
import { vi } from 'vitest';

export class Disposable {
  constructor(private readonly fn: () => void = () => {}) {}
  dispose(): void {
    this.fn();
  }
}

export class EventEmitter<T = void> {
  private listeners: Array<(e: T) => void> = [];
  readonly event = (listener: (e: T) => void): Disposable => {
    this.listeners.push(listener);
    return new Disposable(() => {
      this.listeners = this.listeners.filter((l) => l !== listener);
    });
  };
  fire(e?: T): void {
    for (const l of [...this.listeners]) l(e as T);
  }
  dispose = vi.fn(() => {
    this.listeners = [];
  });
}

export class ThemeIcon {
  constructor(
    public readonly id: string,
    public readonly color?: ThemeColor,
  ) {}
}
export class ThemeColor {
  constructor(public readonly id: string) {}
}
export class MarkdownString {
  constructor(public value = '') {}
}
export const TreeItemCollapsibleState = { None: 0, Collapsed: 1, Expanded: 2 } as const;
export class TreeItem {
  description?: string | boolean;
  tooltip?: string | MarkdownString;
  iconPath?: ThemeIcon;
  contextValue?: string;
  command?: { command: string; title: string; arguments?: unknown[] };
  constructor(
    public label: string,
    public collapsibleState: number = TreeItemCollapsibleState.None,
  ) {}
}
export class Uri {
  constructor(public readonly fsPath: string) {}
  static file(p: string): Uri {
    return new Uri(p);
  }
  get scheme(): string {
    return 'file';
  }
}

/** Mutable world the fake reads from; reset between tests with resetFakeState(). */
export const fakeState = {
  folders: [{ uri: Uri.file('/workspace'), name: 'workspace' }] as
    Array<{ uri: Uri; name: string }> | undefined,
  config: {} as Record<string, unknown>,
};
export function resetFakeState(): void {
  fakeState.folders = [{ uri: Uri.file('/workspace'), name: 'workspace' }];
  fakeState.config = {};
}

export function createFakeVscode() {
  return {
    Disposable,
    EventEmitter,
    ThemeIcon,
    ThemeColor,
    MarkdownString,
    TreeItem,
    TreeItemCollapsibleState,
    Uri,
    workspace: {
      get workspaceFolders() {
        return fakeState.folders;
      },
      getConfiguration: () => ({
        get: <T>(key: string, dflt?: T): T | undefined =>
          key in fakeState.config ? (fakeState.config[key] as T) : dflt,
      }),
    },
  };
}
