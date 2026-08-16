// Background script
browser.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet White Label Admin Extension installed');
  browser.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:8082' });
});

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, transactions: 0, kyc: 0, status: 'OK' });
  }
  return true;
});
