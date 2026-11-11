/**
 * URL policy for values that come from outside the application.
 *
 * Transaction hashes and explorer bases are rendered as links after a send.
 * Escaping alone is not enough for an href: a `javascript:` URL survives HTML
 * escaping, so every link target is parsed and restricted to http(s) here.
 */

const ALLOWED_PROTOCOLS = ['http:', 'https:']

/** Returns the URL when it is a well-formed http(s) address, otherwise null. */
export function safeExternalUrl(raw: unknown, base?: string): string | null {
  if (typeof raw !== 'string') return null
  const trimmed = raw.trim()
  if (!trimmed) return null
  // Browsers strip control characters before resolving a scheme.
  if (/[\u0000-\u001f\u007f]/.test(trimmed)) return null

  const origin =
    base ?? (typeof window !== 'undefined' ? window.location.origin : 'https://localhost')

  let parsed: URL
  try {
    parsed = new URL(trimmed, origin)
  } catch {
    return null
  }

  if (!ALLOWED_PROTOCOLS.includes(parsed.protocol)) return null
  return parsed.toString()
}

/** Builds an explorer link from a base URL and a 32-byte transaction hash. */
export function explorerTxUrl(explorer: unknown, txHash: unknown): string | null {
  const base = safeExternalUrl(explorer)
  if (!base) return null
  if (typeof txHash !== 'string' || !/^0x[0-9a-fA-F]{64}$/.test(txHash.trim())) return null
  return `${base.replace(/\/+$/, '')}/tx/${txHash.trim()}`
}
