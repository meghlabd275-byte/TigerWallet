// TigerAdmin - Chrome Extension Popup Script

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
    chrome.storage.local.set({ autoApproveKYC: isActive });
  });
  
  // View All Button
  document.getElementById('viewAll').addEventListener('click', function() {
    chrome.tabs.create({ url: 'https://admin.tigerwallet.io/users' });
  });
  
  // Load saved settings
  chrome.storage.local.get(['theme', 'autoApproveKYC'], function(result) {
    if (result.theme === 'light') {
      themeToggle.classList.remove('active');
    }
    if (result.autoApproveKYC) {
      autoApprove.classList.add('active');
    }
  });
});

// Background Service Worker
chrome.runtime.onInstalled.addListener(function() {
  chrome.storage.local.set({
    theme: 'dark',
    autoApproveKYC: false,
    adminToken: null,
    pendingUsers: [],
    pendingTransactions: []
  });
});

// Handle messages
chrome.runtime.onMessage.addListener(function(request, sender, sendResponse) {
  if (request.action === 'getAdminStats') {
    sendResponse({
      totalUsers: 12450,
      totalVolume: '$45.2M',
      pendingKYC: 89,
      systemHealth: '99.9%'
    });
  }
  return true;
});
