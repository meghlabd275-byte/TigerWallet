// ProjectParty MV3 service worker.
// Chrome exposes `chrome`; Firefox exposes `browser`.
const ext = typeof browser !== 'undefined' ? browser : chrome;

const DEFAULT_WEB_URL = 'http://localhost:8107';

ext.runtime.onInstalled.addListener(async () => {
  const { pp_web_url: existing } = await ext.storage.local.get('pp_web_url');
  if (!existing) await ext.storage.local.set({ pp_web_url: DEFAULT_WEB_URL });
});

// A user-configured remote host is an optional permission; request it the first
// time the popup talks to a non-local URL, otherwise MV3 blocks the fetch.
ext.runtime.onMessage.addListener((message, _sender, sendResponse) => {
  if (message && message.action === 'ensureHostPermission' && message.origin) {
    ext.permissions
      .contains({ origins: [message.origin + '/*'] })
      .then(async (has) => {
        if (has) return sendResponse({ granted: true });
        const granted = await ext.permissions.request({ origins: [message.origin + '/*'] });
        sendResponse({ granted });
      })
      .catch(() => sendResponse({ granted: false }));
    return true;
  }
});
