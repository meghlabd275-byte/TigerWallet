// Background script
browser.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet Admin Console Extension installed');
  browser.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:3002' });
});

browser.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, tokens: 0, kyc: 0, status: 'OK' });
  }
  return true;
});
