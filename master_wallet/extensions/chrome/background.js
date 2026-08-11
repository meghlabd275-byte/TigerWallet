// TigerMasterWallet - Background Service Worker

// Handle extension installation
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('TigerMasterWallet installed');
    initializeStorage();
  }
});

// Initialize default storage
function initializeStorage() {
  chrome.storage.local.set({
    theme: 'dark',
    autoApprove: false,
    isLocked: true,
    masterAddress: '0x742d35Cc6634C0532925a3b844Bc9e7595f',
    subWallets: [],
    pendingTransactions: [],
    autoSignRules: []
  });
}

// Handle messages from popup and content scripts
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  switch (request.action) {
    case 'getMasterWalletInfo':
      getMasterWalletInfo(sendResponse);
      return true;
      
    case 'approveTransaction':
      approveTransaction(request.txHash, sendResponse);
      return true;
      
    case 'rejectTransaction':
      rejectTransaction(request.txHash, request.reason, sendResponse);
      return true;
      
    case 'createSubWallet':
      createSubWallet(request.name, request.chain, sendResponse);
      return true;
      
    case 'getAutoSignRules':
      getAutoSignRules(sendResponse);
      return true;
      
    case 'toggleAutoSignRule':
      toggleAutoSignRule(request.ruleId, sendResponse);
      return true;
      
    default:
      sendResponse({ error: 'Unknown action' });
  }
});

// API Functions (would connect to real backend in production)
function getMasterWalletInfo(callback) {
  chrome.storage.local.get(['masterAddress', 'subWallets', 'pendingTransactions'], (result) => {
    callback({
      masterAddress: result.masterAddress || '0x742d...12eB3',
      subWalletCount: (result.subWallets || []).length,
      totalVolume: '$12.5M',
      pendingTxCount: (result.pendingTransactions || []).length,
      isLocked: result.isLocked
    });
  });
}

function approveTransaction(txHash, callback) {
  // Simulate API call
  setTimeout(() => {
    chrome.storage.local.get('pendingTransactions', (result) => {
      const pending = result.pendingTransactions || [];
      const updated = pending.filter(tx => tx.hash !== txHash);
      chrome.storage.local.set({ pendingTransactions: updated });
      callback({ success: true, txHash });
    });
  }, 500);
}

function rejectTransaction(txHash, reason, callback) {
  setTimeout(() => {
    chrome.storage.local.get('pendingTransactions', (result) => {
      const pending = result.pendingTransactions || [];
      const updated = pending.filter(tx => tx.hash !== txHash);
      chrome.storage.local.set({ pendingTransactions: updated });
      callback({ success: true, txHash, reason });
    });
  }, 500);
}

function createSubWallet(name, chain, callback) {
  setTimeout(() => {
    const newWallet = {
      id: 'wallet_' + Date.now(),
      name,
      chain,
      address: '',  // address derived by the canonical wallet-api backend
      balance: '0',
      status: 'Active'
    };
    
    chrome.storage.local.get('subWallets', (result) => {
      const wallets = result.subWallets || [];
      wallets.push(newWallet);
      chrome.storage.local.set({ subWallets: wallets });
      callback({ success: true, wallet: newWallet });
    });
  }, 500);
}

function getAutoSignRules(callback) {
  chrome.storage.local.get('autoSignRules', (result) => {
    callback(result.autoSignRules || []);
  });
}

function toggleAutoSignRule(ruleId, callback) {
  chrome.storage.local.get('autoSignRules', (result) => {
    const rules = result.autoSignRules || [];
    const updated = rules.map(rule => {
      if (rule.id === ruleId) {
        return { ...rule, enabled: !rule.enabled };
      }
      return rule;
    });
    chrome.storage.local.set({ autoSignRules: updated });
    callback({ success: true });
  });
}

// Badge update for pending transactions
function updateBadge() {
  chrome.storage.local.get('pendingTransactions', (result) => {
    const count = (result.pendingTransactions || []).length;
    if (count > 0) {
      chrome.action.setBadgeText({ text: count.toString() });
      chrome.action.setBadgeBackgroundColor({ color: '#ff9800' });
    } else {
      chrome.action.setBadgeText({ text: '' });
    }
  });
}

// Periodically check for pending transactions
setInterval(updateBadge, 30000);
updateBadge();
