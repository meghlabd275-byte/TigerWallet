// WebExtension API compatibility shim.
// Firefox MV2 exposes the promise-based `browser.*` namespace but not
// `chrome.*` promises; Chromium MV3 exposes `chrome.*` (with promises in
// service workers) but not `browser.*`. Alias whichever is present so the
// same source runs in both engines without per-browser code copies.
if (typeof browser !== "undefined" && typeof chrome === "undefined") {
  // eslint-disable-next-line no-global-assign
  chrome = browser;
} else if (typeof chrome !== "undefined" && typeof browser === "undefined") {
  // eslint-disable-next-line no-global-assign
  browser = chrome;
}
// Background script
chrome.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet White Label Admin Extension installed');
  chrome.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:8456' });
});

// The popup lists the governance surface; when it asks for a summary we read
// the real tenant-scoped stats endpoint rather than reporting invented zeros.
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    (async () => {
      const { token, apiUrl } = await chrome.storage.local.get(['token', 'apiUrl']);
      const base = apiUrl || 'http://localhost:8456';
      if (!token) return sendResponse({ error: 'not_authenticated' });
      try {
        const res = await fetch(`${base}/api/v1/admin/stats`, {
          headers: { Authorization: `Bearer ${token}` }
        });
        const text = await res.text();
        let body = null;
        try { body = text ? JSON.parse(text) : null; } catch (_) { body = null; }
        if (!res.ok) {
          return sendResponse({ error: (body && body.error) || `HTTP ${res.status}` });
        }
        sendResponse(body || { error: 'No data returned by the WL admin backend.' });
      } catch (e) {
        sendResponse({ error: 'Failed to reach WL admin backend.' });
      }
    })();
    return true;
  }
  return true;
});
