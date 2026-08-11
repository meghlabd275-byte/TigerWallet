// Background service worker for the UserWallet extension.
chrome.runtime.onInstalled.addListener(() => {
  chrome.storage.local.get('theme', (res) => {
    if (!res.theme) chrome.storage.local.set({ theme: 'light' });
  });
});
