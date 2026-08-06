// Background script
browser.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet Master Admin Extension installed');
  browser.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:3000' });
});

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, transactions: 0, system: 'OK', admins: 0 });
  }
  return true;
});
