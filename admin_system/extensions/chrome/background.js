// Background service worker
chrome.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet Admin System Extension installed');
  chrome.storage.local.set({ darkMode: false, apiUrl: 'http://localhost:8090' });
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    sendResponse({ users: 0, configs: 0, auditLogs: 0, status: 'OK' });
  }
  return true;
});
