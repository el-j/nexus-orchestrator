import { describe, it, expect } from 'vitest';
import { shortPath } from './path';

describe('shortPath', () => {
  it.each([
    ['/Users/me/code/app', 'code/app'],
    ['/app', 'app'],
    ['relative/dir/name', 'dir/name'],
    ['/', ''],
    ['', ''],
  ])('%s -> %s', (input, want) => expect(shortPath(input)).toBe(want));
});
