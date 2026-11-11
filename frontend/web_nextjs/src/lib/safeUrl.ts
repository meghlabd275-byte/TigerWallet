/**
 * URL policy for values that come from outside the application.
 *
 * dApp entries, launchpad projects and NFT metadata are authored by third
 * parties. Rendering one of their fields as a bare `href` is unsafe: a
 * `javascript:` URL runs in the wallet origin when clicked, and any other
 * scheme can turn the page into a redirector. Callers pass the raw value
 * through `safeExternalUrl` and render nothing when it returns null.
 */

const ALLOWED_PROTOCOLS = ['http:', 'https:']

/**
 * Returns the URL when it is a well-formed http(s) address, otherwise null.
 *
 * Values without a scheme are resolved against `base` so relative links keep
 * working.
 */
export function safeExternalUrl(raw: unknown, base?: string): string | null {
  if (typeof raw !== 'string') return null
  const trimmed = raw.trim()
  if (!trimmed) return null

  // Browsers strip these before resolving a scheme, which is how a payload
  // such as "java\nscript:" slips past a naive prefix check.
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

/** Returns the URL only when it is https, for links to third-party sites. */
export function safeHttpsUrl(raw: unknown): string | null {
  const value = safeExternalUrl(raw)
  if (!value) return null
  try {
    if (new URL(value).protocol !== 'https:') return null
  } catch {
    return null
  }
  return value
}

/** Builds an explorer link, returning null unless both parts are safe. */
export function explorerTxUrl(explorer: unknown, txHash: unknown): string | null {
  const base = safeExternalUrl(explorer)
  if (!base) return null
  if (typeof txHash !== 'string' || !/^0x[0-9a-fA-F]{64}$/.test(txHash.trim())) return null
  return `${base.replace(/\/+$/, '')}/tx/${txHash.trim()}`
}
