/**
 * Helpers for rendering values that originate outside the application.
 *
 * KYC document URLs are submitted by end users and then shown to operators as
 * clickable links. A bare `href` on such a value is unsafe in two ways:
 *
 *  - `javascript:` and `data:` URLs execute in the origin that renders them, so
 *    a submitted `javascript:...` link runs script inside the admin console
 *    when an operator clicks "view".
 *  - A value like `//evil.example` or a full external URL turns the admin
 *    console into a redirector, which is what phishing campaigns want.
 *
 * Only http(s) URLs that point at the same origin as the console are allowed to
 * be rendered as links. Everything else is shown as inert text.
 */

/** Returns the URL when it is a same-origin http(s) URL, otherwise null. */
export function safeDocumentURL(raw: unknown): string | null {
  if (typeof raw !== 'string') return null;
  const trimmed = raw.trim();
  if (!trimmed) return null;

  let parsed: URL;
  try {
    parsed = new URL(trimmed, window.location.origin);
  } catch {
    return null;
  }

  if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return null;
  if (parsed.origin !== window.location.origin) return null;

  return parsed.pathname + parsed.search + parsed.hash;
}

/**
 * Renders a stored document reference. Returns a link for a same-origin http(s)
 * URL and plain text for anything else, so an unsafe value can never be
 * clicked through to script execution or to an attacker-controlled host.
 */
export function DocumentLink({ url }: { url: unknown }) {
  const safe = safeDocumentURL(url);
  if (!safe) {
    return url ? <span className="text-xs opacity-60">unsafe link blocked</span> : null;
  }
  return (
    <a
      href={safe}
      target="_blank"
      rel="noopener noreferrer nofollow"
      className="ml-2 text-blue-600 hover:underline"
    >
      view
    </a>
  );
}
