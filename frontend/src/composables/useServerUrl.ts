import { getServerAddr } from '../types/wails'

// Module-level cache: resolved once per app session.
let cachedUrl: string | null = null

/**
 * Resolves the base HTTP URL of the embedded API server.
 * On first call fetches from the Wails binding (or falls back to the
 * default dev address); subsequent calls return the cached value.
 */
export async function resolveServerUrl(): Promise<string> {
  if (cachedUrl !== null) return cachedUrl
  cachedUrl = await getServerAddr()
  return cachedUrl
}
