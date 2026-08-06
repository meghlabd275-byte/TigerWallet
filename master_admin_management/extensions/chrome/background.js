// Background service worker
chrome.runtime.onInstalled.addListener(() => {
  console.log('TigerWallet Master Admin Extension installed');
  
  // Set default values
  chrome.storage.local.set({
    darkMode: false,
    apiUrl: 'http://localhost:9091'
  });
});

// Handle messages from popup
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'getStats') {
    // Return simulated stats
    sendResponse({
      whiteLabels: 0,
      users: 0,
      transactions: 0,
      pending: 0
    });
  }
  return true;
});
