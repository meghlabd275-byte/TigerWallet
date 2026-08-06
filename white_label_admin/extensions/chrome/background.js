// Background script
chrome.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet White Label Admin Extension installed');
  chrome.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:3001' });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, transactions: 0, kyc: 0, status: 'OK' });
  }
  return true;
});
