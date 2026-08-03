// TigerMasterWallet - Chrome Extension Popup Script

document.addEventListener('DOMContentLoaded', function() {
  // Theme Toggle
  const themeToggle = document.getElementById('themeToggle');
  themeToggle.addEventListener('click', function() {
    this.classList.toggle('active');
    const isDark = this.classList.contains('active');
    chrome.storage.local.set({ theme: isDark ? 'dark' : 'light' });
  });
  
  // Auto Approve Toggle
  const autoApprove = document.getElementById('autoApprove');
  autoApprove.addEventListener('click', function() {
    this.classList.toggle('active');
    const isActive = this.classList.contains('active');
    chrome.storage.local.set({ autoApprove: isActive });
  });
  
  // View All Button
  document.getElementById('viewAll').addEventListener('click', function() {
    chrome.tabs.create({ url: 'https://master.tigerwallet.io/wallets' });
  });
  
  // Load saved settings
  chrome.storage.local.get(['theme', 'autoApprove'], function(result) {
    if (result.theme === 'light') {
      themeToggle.classList.remove('active');
    }
    if (result.autoApprove) {
      autoApprove.classList.add('active');
    }
  });
});

// Background Service Worker
chrome.runtime.onInstalled.addListener(function() {
  chrome.storage.local.set({
    theme: 'dark',
    autoApprove: false,
    wallets: [],
    pendingTransactions: []
  });
});

// Handle messages from content scripts
chrome.runtime.onMessage.addListener(function(request, sender, sendResponse) {
  if (request.action === 'getMasterWalletInfo') {
    sendResponse({
      connected: true,
      masterAddress: '0x742d35Cc6634C0532925a3b844Bc9e7595f',
      subWalletCount: 15,
      totalVolume: '$12.5M'
    });
  }
  return true;
});
