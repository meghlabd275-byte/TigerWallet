// Background service worker for UserWallet extension
browser.runtime.onInstalled.addListener(() => {
  // Set default theme
  browser.storage.local.set({ theme: 'light' });
});

// Listen for messages from popup
browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getTheme') {
    browser.storage.local.get('theme').then(result => {
      sendResponse({ theme: result.theme });
    });
    return true;
  }
});
