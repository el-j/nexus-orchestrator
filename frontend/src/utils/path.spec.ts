import { describe, it, expect } from 'vitest';
import { shortPath, baseName } from './path';

describe('shortPath', () => {
  it.each([
    ['/Users/me/code/app', 'code/app'],
    ['/app', 'app'],
    ['relative/dir/name', 'dir/name'],
    ['/', ''],
    ['', ''],
  ])('%s -> %s', (input, want) => expect(shortPath(input)).toBe(want));
});

describe('baseName', () => {
  it.each([
    ['/Users/me/code/app', 'app'],
    ['/app', 'app'],
    ['relative/dir/name', 'name'],
    ['/', '/'],
    ['', ''],
  ])('%s -> %s', (input, want) => expect(baseName(input)).toBe(want));
});
