// Background script
browser.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet Admin Services Extension installed');
  browser.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:9090' });
});

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ services: 0, health: 'OK', uptime: '0h', errors: 0 });
  }
  return true;
});
