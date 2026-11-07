/**
 * errorText.ts: the text of a caught value.
 */

/** The message of an Error, or the string form of anything else that was thrown. */
export function errorText(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}
