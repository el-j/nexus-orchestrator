import { describe, expect, it } from 'vitest';
import { errorText } from './errorText';

describe('errorText', () => {
  it('uses the message of an Error', () => {
    expect(errorText(new Error('down'))).toBe('down');
  });
  it('turns anything else into a string', () => {
    expect(errorText('plain')).toBe('plain');
    expect(errorText(42)).toBe('42');
    expect(errorText(undefined)).toBe('undefined');
  });
});
