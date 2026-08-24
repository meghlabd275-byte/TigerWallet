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
  chrome.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:8082' });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, transactions: 0, kyc: 0, status: 'OK' });
  }
  return true;
});
